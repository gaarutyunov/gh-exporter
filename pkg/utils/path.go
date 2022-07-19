package utils

import (
	"os/user"
	"path"
	"strings"
)

var homeDir string

func init() {
	usr, err := user.Current()
	if err != nil {
		homeDir = ""
	}

	homeDir = usr.HomeDir
}

func ExpandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		return path.Join(homeDir, strings.TrimLeft(p, "~"))
	}

	return p
}
