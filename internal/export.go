package internal

import (
	"gh-exporter/pkg/gh"
	"gh-exporter/pkg/utils"
	"github.com/cheggaaa/pb/v3"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v45/github"
	"github.com/spf13/cobra"
	"github.com/xxjwxc/gowp/workpool"
	"os"
)

func Export(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client := gh.NewClient(ctx)

	query, err := cmd.PersistentFlags().GetString("query")
	if err != nil {
		return err
	}

	res, _, err := client.Search.Repositories(ctx, query, &github.SearchOptions{
		TextMatch: false,
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: 1,
		},
	})
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

	pattern, err := cmd.PersistentFlags().GetString("pattern")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(outDir, os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	pool := workpool.New(10)
	perPage := 100
	totalPages := res.GetTotal() / perPage

	bar := pb.StartNew(res.GetTotal())

	for i := 0; i <= totalPages; i++ {
		ii := i

		pool.Do(func() error {
			if err := client.Wait(ctx); err != nil {
				return err
			}

			res, _, err := client.Search.Repositories(ctx, query, &github.SearchOptions{
				TextMatch: false,
				ListOptions: github.ListOptions{
					Page:    ii,
					PerPage: perPage,
				},
			})
			if err != nil {
				return err
			}

			pool := workpool.New(10)

			for _, rr := range res.Repositories {
				repository := gh.NewRepo(rr.GetSSHURL(), rr.GetFullName(), outDir, rr.GetSize())

				if ok, err := repository.Exists(); err != nil {
					return err
				} else if ok {
					bar.AddTotal(-1)

					continue
				}

				pool.Do(func() error {
					if err := repository.Clone(ctx, publicKey, pattern); err != nil {
						return err
					}

					bar.Increment()

					return err
				})
			}

			return pool.Wait()
		})
	}

	if err := pool.Wait(); err != nil {
		return err
	}

	bar.Finish()

	return nil
}
