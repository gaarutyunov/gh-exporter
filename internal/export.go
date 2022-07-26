package internal

import (
	"bufio"
	"github.com/cheggaaa/pb/v3"
	"github.com/gaarutyunov/gh-exporter/pkg/gh"
	"github.com/gaarutyunov/gh-exporter/pkg/utils"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/xxjwxc/gowp/workpool"
	"os"
	"strings"
)

func Export(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	concurrency, err := cmd.PersistentFlags().GetInt("concurrency")
	if err != nil {
		return err
	}

	sshPath, err := cmd.PersistentFlags().GetString("identity")
	if err != nil {
		return err
	}
	sshPath = utils.ExpandPath(sshPath)
	publicKey, err := ssh.NewPublicKeysFromFile("git", sshPath, "")
	if err != nil {
		return err
	}

	outDir, err := cmd.PersistentFlags().GetString("out")
	if err != nil {
		return err
	}
	outDir = utils.ExpandPath(outDir)
	if err = os.MkdirAll(outDir, os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	planFile, err := cmd.PersistentFlags().GetString("file")
	if err != nil {
		return err
	}
	planFile = utils.ExpandPath(planFile)

	searchFile, err := cmd.PersistentFlags().GetString("search")
	if err != nil {
		return err
	}
	searchFile = utils.ExpandPath(searchFile)

	pattern, err := cmd.PersistentFlags().GetString("pattern")
	if err != nil {
		return err
	}

	fsearch, err := os.Open(searchFile)
	if err != nil {
		return err
	}
	total, err := utils.LineCounter(fsearch)
	if err != nil {
		return err
	}
	if err = fsearch.Close(); err != nil {
		return err
	}

	bar := pb.StartNew(total)

	fin, err := os.Open(planFile)
	if err != nil {
		return err
	}
	defer func(fin *os.File) {
		err = fin.Close()
	}(fin)

	scanner := bufio.NewScanner(fin)
	var group []*gh.Repo
	var remainder []*gh.Repo
	var isRemainder bool

	for scanner.Scan() {
		if scanner.Text() == "" {
			pool := workpool.New(concurrency)

			for _, rr := range group {
				repository := rr

				if ok, err := repository.Exists(); err != nil {
					return err
				} else if ok {
					bar.AddTotal(-1)
					continue
				}

				pool.Do(func() error {
					defer bar.Increment()
					if err := repository.CloneMem(ctx, publicKey, pattern); err != nil {
						return err
					}

					return err
				})
			}

			if err = pool.Wait(); err != nil {
				return err
			}

			group = []*gh.Repo{}
		} else if scanner.Text() == "---" {
			isRemainder = true
		} else {
			vals := strings.Split(scanner.Text(), ";")
			fullName, sshUrl, size := vals[0], vals[1], vals[2]

			repo := gh.NewRepo(sshUrl, fullName, outDir, cast.ToUint64(size))
			if isRemainder {
				remainder = append(remainder, repo)
			} else {
				group = append(group, repo)
			}
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	pool := workpool.New(concurrency)

	for _, rr := range remainder {
		repository := rr

		if ok, err := repository.Exists(); err != nil {
			return err
		} else if ok {
			bar.AddTotal(-1)
			continue
		}

		pool.Do(func() error {
			defer bar.Increment()
			if err := repository.CloneFS(ctx, publicKey, pattern); err != nil {
				return err
			}

			return err
		})
	}

	if err = pool.Wait(); err != nil {
		return err
	}

	bar.Finish()

	return nil
}
