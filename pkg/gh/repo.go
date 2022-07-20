package gh

import (
	"context"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/shirou/gopsutil/v3/mem"
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
	mem      billy.Filesystem
	os       billy.Filesystem
}

func NewRepo(sshURL string, fullName string, cloneDir string, size int) *Repo {
	repoDir := path.Join(cloneDir, fullName)

	return &Repo{
		sshURL:   sshURL,
		fullName: fullName,
		repoDir:  repoDir,
		size:     uint64(size) * uint64(cache.KiByte),
		mem:      memfs.New(),
		os:       osfs.New(repoDir),
	}
}

func (r *Repo) Clone(ctx context.Context, sshKey *ssh.PublicKeys, pattern string) error {
	v, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	if v.Available < r.size+1*uint64(cache.GiByte) {
		return r.CloneFS(ctx, sshKey, pattern)
	}

	return r.CloneMem(ctx, sshKey, pattern)
}

func (r *Repo) CloneFS(ctx context.Context, sshKey *ssh.PublicKeys, pattern string) error {
	_, err := git.CloneContext(ctx, filesystem.NewStorage(r.os, cache.NewObjectLRU(128*cache.MiByte)), r.os, &git.CloneOptions{
		Auth: sshKey,
		URL:  r.sshURL,
	})
	if err != nil {
		return err
	}

	var read func(dir string) error

	read = func(dir string) error {
		files, err := r.os.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, file := range files {
			fullName := file.Name()

			if dir != "/" {
				fullName = path.Join(dir, file.Name())
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
				if err := r.os.Remove(fullName); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return read("/")
}

func (r *Repo) CloneMem(ctx context.Context, sshKey *ssh.PublicKeys, pattern string) error {
	_, err := git.CloneContext(ctx, memory.NewStorage(), r.mem, &git.CloneOptions{
		Auth: sshKey,
		URL:  r.sshURL,
	})
	if err != nil {
		return err
	}

	var read func(dir string) error

	read = func(dir string) error {
		files, err := r.mem.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, file := range files {
			fullName := file.Name()

			if dir != "/" {
				fullName = path.Join(dir, file.Name())
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

			src, err := r.mem.Open(fullName)
			if err != nil {
				return err
			}

			dst, err := r.os.Create(fullName)
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
