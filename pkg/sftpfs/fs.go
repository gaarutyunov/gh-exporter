package sftpfs

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/helper/chroot"
	"github.com/go-git/go-billy/v5/helper/temporal"
	"github.com/pkg/sftp"
	"os"
	"path/filepath"
	"sync"
)

type Fs struct {
	*sftp.Client
	root string
}

type fileWrapper struct {
	*sftp.File
	mx sync.Mutex
}

func (f *fileWrapper) Lock() error {
	f.mx.Lock()
	return nil
}

func (f *fileWrapper) Unlock() error {
	f.mx.Unlock()
	return nil
}

func (s *Fs) Create(filename string) (billy.File, error) {
	dir := filepath.Dir(filename)
	err := s.MkdirAll(dir, os.ModeDir)
	if err != nil {
		return nil, err
	}
	f, err := s.Client.Create(filename)
	if err != nil {
		return nil, err
	}
	return &fileWrapper{File: f}, nil
}

func (s *Fs) Open(filename string) (billy.File, error) {
	f, err := s.Client.Open(filename)
	if err != nil {
		return nil, err
	}
	return &fileWrapper{File: f}, nil
}

func (s *Fs) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	f, err := s.Client.OpenFile(filename, flag)
	if err != nil {
		return nil, err
	}
	return &fileWrapper{File: f}, nil
}

func (s *Fs) MkdirAll(filename string, perm os.FileMode) error {
	err := s.Client.MkdirAll(filename)
	if err != nil {
		return err
	}

	return nil
}

func (s *Fs) Readlink(link string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func New(client *sftp.Client) billy.Filesystem {
	return temporal.New(chroot.New(&Fs{Client: client}, ""), "")
}
