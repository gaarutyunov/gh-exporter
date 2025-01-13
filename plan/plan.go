package plan

import (
	"bufio"
	"github.com/gaarutyunov/gh-exporter/gh"
	"iter"
	"os"
	"strings"
)

type File struct {
	Bins      [][]gh.RepoInfo
	Remainder []gh.RepoInfo
}

func New(bins [][]gh.RepoInfo, remainder []gh.RepoInfo) File {
	return File{Bins: bins, Remainder: remainder}
}

func (f File) Total(skipRemainder, onlyRemainder bool) (n int) {
	if !onlyRemainder {
		for _, bin := range f.Bins {
			for range bin {
				n++
			}
		}
	}

	if skipRemainder {
		return
	}

	n += len(f.Remainder)

	return
}

func Open(path string) (file File, err error) {
	fi, err := os.Open(path)
	if err != nil {
		return
	}
	defer fi.Close()

	scanner := bufio.NewScanner(fi)

	file.Bins = [][]gh.RepoInfo{}
	file.Remainder = []gh.RepoInfo{}

	var isRemainder bool
	var group []gh.RepoInfo

	for scanner.Scan() {
		if scanner.Text() == "" {
			file.Bins = append(file.Bins, group)

			group = []gh.RepoInfo{}
		} else if scanner.Text() == "---" {
			isRemainder = true
		} else {
			repo, err := gh.RepoInfoFromString(scanner.Text())
			if err != nil {
				return file, err
			}

			if isRemainder {
				file.Remainder = append(file.Remainder, repo)
			} else {
				group = append(group, repo)
			}
		}
	}

	if scanner.Err() != nil {
		err = scanner.Err()
	}

	return
}

func (f File) Iter(skipRemainder, onlyRemainder bool) iter.Seq2[[]gh.RepoInfo, bool] {
	return func(yield func([]gh.RepoInfo, bool) bool) {
		if !onlyRemainder {
			for _, bin := range f.Bins {
				if !yield(bin, false) {
					return
				}
			}
		}

		if skipRemainder {
			return
		}

		if !yield(f.Remainder, true) {
			return
		}
	}
}

func (f File) String() string {
	var builder strings.Builder

	for _, bin := range f.Bins {
		for _, repo := range bin {
			builder.WriteString(repo.String())
			builder.WriteString("\n")
		}

		builder.WriteString("\n")
	}

	if len(f.Remainder) > 0 {
		builder.WriteString("---\n")
	}

	for _, repo := range f.Remainder {
		builder.WriteString(repo.String())
		builder.WriteString("\n")
	}

	return builder.String()
}
