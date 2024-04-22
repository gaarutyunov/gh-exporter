package gh

import (
	"context"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/chroot"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Repo struct {
	sshURL   string
	fullName string
	repoDir  string
	sha      string
	size     uint64
}

var Delimiter = "."

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

func (r *Repo) CloneFS(ctx context.Context, sshKey *ssh.PublicKeys, pattern string, outFs billy.Filesystem) error {
	outFs = chroot.New(outFs, r.repoDir)

	rr, err := git.CloneContext(ctx, filesystem.NewStorage(outFs, cache.NewObjectLRU(128*cache.MiByte)), outFs, &git.CloneOptions{
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
				return err
			}
		}
	}

	var read func(dir string) (count int, err error)

	read = func(dir string) (count int, err error) {
		files, err := outFs.ReadDir(dir)
		if err != nil {
			return 0, err
		}
		count = len(files)

		for _, file := range files {
			fullName := file.Name()

			if dir != "/" {
				fullName = path.Join(dir, file.Name())
			}

			if file.IsDir() {
				if n, err := read(fullName); err != nil {
					return 0, err
				} else if n == 0 {
					if err := outFs.Remove(fullName); err != nil {
						return 0, err
					}
					count -= 1
				}
				continue
			}

			match, err := filepath.Match(pattern, file.Name())
			if err != nil {
				return 0, err
			}
			if !match {
				if err := outFs.Remove(fullName); err != nil {
					return 0, err
				}
				count -= 1
			}
		}

		return count, err
	}

	_, err = read("/")
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) CloneMem(ctx context.Context, sshKey *ssh.PublicKeys, pattern string, outFs billy.Filesystem) error {
	fs := memfs.New()
	outFs = chroot.New(outFs, r.repoDir)

	rr, err := git.CloneContext(ctx, memory.NewStorage(), fs, &git.CloneOptions{
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

	var read func(dir string) error

	read = func(dir string) error {
		files, err := fs.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, file := range files {
			fullName := file.Name()

			if dir != "/" {
				fullName = path.Join(dir, file.Name())
			}

			if file.Mode()&os.ModeSymlink != 0 {
				continue
			}

			if file.IsDir() {
				if err = read(fullName); err != nil {
					return err
				}
				continue
			}
			match, err := filepath.Match(pattern, file.Name())
			if err != nil {
				return err
			}
			if !match {
				continue
			}

			src, err := fs.Open(fullName)
			if err != nil {
				return err
			}

			dst, err := outFs.Create(fullName)
			if err != nil {
				return err
			}

			if _, err = io.Copy(dst, src); err != nil {
				return err
			}

			if err := dst.Close(); err != nil {
				return err
			}

			if err := src.Close(); err != nil {
				return err
			}
		}

		return nil
	}

	return read("/")
}

func (r *Repo) Exists() (bool, error) {
	if _, err := os.Stat(r.repoDir); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
