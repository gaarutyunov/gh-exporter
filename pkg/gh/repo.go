package gh

import (
	"context"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"io"
	"os"
	"path"
	"path/filepath"
)

type Repo struct {
	sshURL   string
	fullName string
	repoDir  string
	size     uint64
}

func (r *Repo) SshURL() string {
	return r.sshURL
}

func (r *Repo) FullName() string {
	return r.fullName
}

func (r *Repo) Size() uint64 {
	return r.size
}

func NewRepo(sshURL string, fullName string, rootDir string, size uint64) *Repo {
	cloneDir := path.Join(rootDir, fullName)

	return &Repo{
		sshURL:   sshURL,
		fullName: fullName,
		repoDir:  cloneDir,
		size:     size * uint64(cache.KiByte),
	}
}

func (r *Repo) CloneFS(ctx context.Context, sshKey *ssh.PublicKeys, pattern string) error {
	fs := osfs.New(r.repoDir)

	_, err := git.CloneContext(ctx, filesystem.NewStorage(fs, cache.NewObjectLRU(128*cache.MiByte)), fs, &git.CloneOptions{
		Auth: sshKey,
		URL:  r.sshURL,
	})
	if err != nil {
		return err
	}

	var read func(dir string) (count int, err error)

	read = func(dir string) (count int, err error) {
		files, err := fs.ReadDir(dir)
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
					if err := fs.Remove(fullName); err != nil {
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
				if err := fs.Remove(fullName); err != nil {
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

func (r *Repo) CloneMem(ctx context.Context, sshKey *ssh.PublicKeys, pattern string) error {
	fs := memfs.New()
	fsDisk := osfs.New(r.repoDir)

	_, err := git.CloneContext(ctx, memory.NewStorage(), fs, &git.CloneOptions{
		Auth: sshKey,
		URL:  r.sshURL,
	})
	if err != nil {
		return err
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

			dst, err := fsDisk.Create(fullName)
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
