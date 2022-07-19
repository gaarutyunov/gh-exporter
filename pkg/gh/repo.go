package gh

import (
	"context"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
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
	mem      billy.Filesystem
	os       billy.Filesystem
}

func NewRepo(sshURL string, fullName string, cloneDir string) *Repo {
	repoDir := path.Join(cloneDir, fullName)

	return &Repo{
		sshURL:   sshURL,
		fullName: fullName,
		repoDir:  repoDir,
		mem:      memfs.New(),
		os:       osfs.New(repoDir),
	}
}

func (r *Repo) Clone(ctx context.Context, sshKey *ssh.PublicKeys, pattern string) error {
	_, err := git.CloneContext(ctx, memory.NewStorage(), r.mem, &git.CloneOptions{
		Auth: sshKey,
		URL:  r.sshURL,
	})
	if err != nil {
		return err
	}

	files, err := r.mem.ReadDir("/")
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		match, err := filepath.Match(pattern, file.Name())
		if err != nil {
			return err
		}
		if !match {
			continue
		}

		src, err := r.mem.Open(file.Name())
		if err != nil {
			return err
		}

		dst, err := r.os.Create(file.Name())
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

	return err
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
