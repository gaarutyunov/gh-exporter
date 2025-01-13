package gh

import (
	"fmt"
	"github.com/spf13/cast"
	"strings"
)

type RepoInfo struct {
	sshURL   string
	fullName string
	repoDir  string
	sha      string
	size     uint64
}

func NewRepoInfo(fullName string, sshURL string, size uint64) RepoInfo {
	return RepoInfo{
		sshURL:   sshURL,
		fullName: fullName,
		repoDir:  strings.Replace(fullName, "/", Delimiter, 1),
		size:     size,
	}
}

func (r RepoInfo) WithSHA(sha string) RepoInfo {
	r.sha = sha

	return r
}

func RepoInfoFromString(s string) (repo RepoInfo, err error) {
	vals := strings.Split(s, ";")
	if len(vals) < 3 {
		err = fmt.Errorf("invalid string format: %s", s)
		return
	}

	repo = NewRepoInfo(vals[0], vals[1], cast.ToUint64(vals[2]))
	if len(vals) > 3 {
		repo = repo.WithSHA(strings.TrimSpace(vals[3]))
	}

	return
}

func (r RepoInfo) SshURL() string {
	return r.sshURL
}

func (r RepoInfo) FullName() string {
	return r.fullName
}

func (r RepoInfo) Owner() string {
	return strings.Split(r.fullName, "/")[0]
}

func (r RepoInfo) Name() string {
	return strings.Split(r.fullName, "/")[1]
}

func (r RepoInfo) SHA() string {
	return r.sha
}

// Size returns the size of the repository in kilobytes.
func (r RepoInfo) Size() uint64 {
	return r.size
}

func (r RepoInfo) Dir() string {
	return r.repoDir
}

func (r RepoInfo) String() string {
	return fmt.Sprintf(
		"%s;%s;%d;%s",
		r.FullName(),
		r.SshURL(),
		r.Size(),
		r.SHA(),
	)
}
