package gh

import (
	"context"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/chroot"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-github/v45/github"
	"github.com/sirupsen/logrus"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Repo struct {
	sshURL   string
	fullName string
	repoDir  string
	sha      string
	size     uint64
	ghRepo   *github.Repository
	mx       sync.Mutex
}

var (
	Delimiter     = "."
	DefaultBranch = "master"
)

func (r *Repo) SshURL() string {
	return r.sshURL
}

func (r *Repo) FullName() string {
	return r.fullName
}

func (r *Repo) SHA() string {
	return r.sha
}

func (r *Repo) Size() uint64 {
	return r.size
}

func (r *Repo) Dir() string {
	return r.repoDir
}

func NewRepo(sshURL string, fullName string, size uint64) *Repo {
	cloneDir := strings.Replace(fullName, "/", Delimiter, 1)

	return &Repo{
		sshURL:   sshURL,
		fullName: fullName,
		repoDir:  cloneDir,
		size:     size * uint64(cache.KiByte),
	}
}

func (r *Repo) SetSHA(sha string) {
	r.sha = sha
}

func (r *Repo) GetGithubRepo(ctx context.Context) *github.Repository {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.ghRepo != nil {
		return r.ghRepo
	}

	client := NewClient(ctx)
	spl := strings.Split(r.fullName, "/")

	repository, _, err := client.Repositories.Get(ctx, spl[0], spl[1])
	if err != nil {
		return nil
	}
	r.ghRepo = repository

	return r.ghRepo
}

func (r *Repo) GetDefaultBranch(ctx context.Context) string {
	repository := r.GetGithubRepo(ctx)
	if repository == nil {
		return DefaultBranch
	}

	branch := repository.GetDefaultBranch()
	if branch == "" {
		branch = repository.GetMasterBranch()
	}
	if branch == "" {
		branch = DefaultBranch
	}

	return branch
}

func (r *Repo) CloneFS(ctx context.Context, sshKey *ssh.PublicKeys, pattern string, outFs billy.Filesystem) error {
	outFs = chroot.New(outFs, r.repoDir)

	branch := r.GetDefaultBranch(ctx)
	logrus.Debugf("started cloning %s from %s into %s", r.FullName(), branch, outFs.Root())

	dot, err := outFs.Chroot(git.GitDirName)
	if err != nil {
		return err
	}

	rr, err := git.CloneContext(ctx, filesystem.NewStorage(dot, cache.NewObjectLRU(128*cache.MiByte)), outFs, &git.CloneOptions{
		Auth:          sshKey,
		URL:           r.sshURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		return err
	}

	if r.sha != "" {
		if w, err := rr.Worktree(); err != nil {
			return err
		} else {
			err := w.Reset(&git.ResetOptions{
				Commit: plumbing.NewHash(r.sha),
				Mode:   git.HardReset,
			})
			if err != nil {
				logrus.Errorf("error resetting %s to %s: %s", r.FullName(), r.SHA(), err)
			}
		}
	}

	err = util.Walk(outFs, outFs.Root(), func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			return nil
		}

		if match, err := filepath.Match(pattern, info.Name()); err != nil {
			return err
		} else if !match {
			if err := outFs.Remove(path); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	logrus.Debugf("finished cloning %s from %s into %s", r.FullName(), branch, outFs.Root())

	return nil
}

func (r *Repo) CloneMem(ctx context.Context, sshKey *ssh.PublicKeys, pattern string, outFs billy.Filesystem) error {
	memFs := memfs.New()
	outFs = chroot.New(outFs, r.repoDir)

	branch := r.GetDefaultBranch(ctx)
	logrus.Debugf("started cloning %s from %s through memory into %s", r.FullName(), branch, outFs.Root())

	rr, err := git.CloneContext(ctx, memory.NewStorage(), memFs, &git.CloneOptions{
		Auth:          sshKey,
		URL:           r.sshURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		return err
	}

	if r.sha != "" {
		if w, err := rr.Worktree(); err != nil {
			return err
		} else {
			err := w.Reset(&git.ResetOptions{
				Commit: plumbing.NewHash(r.sha),
				Mode:   git.HardReset,
			})
			if err != nil {
				logrus.Errorf("error resetting %s to %s: %s", r.FullName(), r.SHA(), err)
			}
		}
	}

	err = util.Walk(memFs, memFs.Root(), func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			return nil
		}

		if match, err := filepath.Match(pattern, info.Name()); err != nil {
			return err
		} else if !match {
			return nil
		}

		src, err := memFs.Open(path)
		if err != nil {
			return err
		}

		dst, err := outFs.Create(path)
		if err != nil {
			return err
		}

		defer func() {
			_ = src.Close()
			_ = dst.Close()
		}()

		_, err = io.Copy(dst, src)

		return err
	})
	if err != nil {
		return err
	}

	logrus.Debugf("finished cloning %s from %s through memory into %s", r.FullName(), branch, outFs.Root())

	return nil
}

func (r *Repo) Exists(fs billy.Filesystem) (bool, error) {
	if _, err := fs.Stat(r.repoDir); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
