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
)

type Repo struct {
	RepoInfo
	ghRepo *github.Repository
}

var (
	Delimiter     = "."
	DefaultBranch = "master"
	Depth         = 1
)

func NewRepo(info RepoInfo, ghRepo *github.Repository) *Repo {
	return &Repo{
		RepoInfo: info,
		ghRepo:   ghRepo,
	}
}

func (r *Repo) SetSHA(sha string) {
	r.RepoInfo = r.RepoInfo.WithSHA(sha)
}

func (r *Repo) GetDefaultBranch() string {
	repository := r.ghRepo
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

	dot, err := outFs.Chroot(git.GitDirName)
	if err != nil {
		return err
	}

	rr, err := git.CloneContext(ctx, filesystem.NewStorage(dot, cache.NewObjectLRU(128*cache.MiByte)), outFs, &git.CloneOptions{
		Auth: sshKey,
		URL:  r.sshURL,
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

	err = util.Walk(outFs, "/", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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

	return nil
}

func (r *Repo) CloneMem(ctx context.Context, sshKey *ssh.PublicKeys, pattern string, outFs billy.Filesystem) error {
	memFs := memfs.New()
	outFs = chroot.New(outFs, r.repoDir)
	storage := memory.NewStorage()

	rr, err := git.CloneContext(ctx, storage, memFs, &git.CloneOptions{
		Auth: sshKey,
		URL:  r.sshURL,
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
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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
