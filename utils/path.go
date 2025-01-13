package utils

import (
	"os"
	"os/user"
	"path"
	"strings"
)

var homeDir string

func init() {
	usr, err := user.Current()
	if err == nil {
		homeDir = usr.HomeDir
	}
}

func ExpandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		return path.Join(homeDir, strings.TrimLeft(p, "~"))
	}

	return p
}

func TryCreate(p string) (fi *os.File, err error) {
	if fi, err = os.OpenFile(p, os.O_RDWR, os.ModePerm); err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if err == nil {
		return
	}

	d := path.Dir(p)

	if err := os.MkdirAll(d, os.ModePerm); err != nil && !os.IsExist(err) {
		return nil, err
	}

	if fi, err = os.Create(p); err != nil && !os.IsExist(err) {
		return nil, err
	}

	return
}
