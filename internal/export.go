package internal

import (
	"bufio"
	"context"
	"github.com/cheggaaa/pb/v3"
	"github.com/gaarutyunov/gh-exporter/pkg/gh"
	"github.com/gaarutyunov/gh-exporter/pkg/utils"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os"
	"strings"
)

func Export(cmd *cobra.Command, args []string) error {
	outDir, err := cmd.PersistentFlags().GetString("out")
	if err != nil {
		return err
	}

	return doExport(cmd.Context(), cmd, args, osfs.New(outDir))
}

func doExport(ctx context.Context, cmd *cobra.Command, args []string, outFs billy.Filesystem) error {
	if err := outFs.MkdirAll(outFs.Root(), os.ModeDir); err != nil && !os.IsExist(err) {
		return err
	}

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
			var wg errgroup.Group
			wg.SetLimit(concurrency)

			for _, rr := range group {
				repository := rr

				if ok, err := repository.Exists(outFs); err != nil {
					return err
				} else if ok {
					bar.AddTotal(-1)
					continue
				}

				wg.Go(func() error {
					defer bar.Increment()
					if err := repository.CloneMem(ctx, publicKey, pattern, outFs); err != nil {
						logrus.Errorf("error for %s: %s", repository.FullName(), err)
					}

					return nil
				})
			}

			if err = wg.Wait(); err != nil {
				return err
			}

			group = []*gh.Repo{}
		} else if scanner.Text() == "---" {
			isRemainder = true
		} else {
			vals := strings.Split(scanner.Text(), ";")
			fullName, sshUrl, size := vals[0], vals[1], vals[2]

			repo := gh.NewRepo(sshUrl, fullName, cast.ToUint64(size))
			if len(vals) > 3 {
				sha := strings.TrimSpace(vals[3])
				repo.SetSHA(sha)
			}
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

	var wg errgroup.Group
	wg.SetLimit(concurrency)

	for _, rr := range remainder {
		repository := rr

		if ok, err := repository.Exists(outFs); err != nil {
			return err
		} else if ok {
			bar.AddTotal(-1)
			continue
		}

		wg.Go(func() error {
			defer bar.Increment()
			if err := repository.CloneFS(ctx, publicKey, pattern, outFs); err != nil {
				logrus.Errorf("error for %s: %s", repository.FullName(), err)
			}

			return err
		})
	}

	if err = wg.Wait(); err != nil {
		return err
	}

	bar.Finish()

	return nil
}
