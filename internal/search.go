package internal

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/gaarutyunov/gh-exporter/pkg/gh"
	"github.com/gaarutyunov/gh-exporter/pkg/utils"
	"github.com/google/go-github/v45/github"
	"github.com/spf13/cobra"
	"github.com/xxjwxc/gowp/workpool"
	"os"
	"sync"
)

func Search(cmd *cobra.Command, args []string) error {
	client := gh.NewClient(cmd.Context())

	query, err := cmd.PersistentFlags().GetString("query")
	if err != nil {
		return err
	}

	res, _, err := client.Search.Repositories(cmd.Context(), query, &github.SearchOptions{
		TextMatch: false,
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: 1,
		},
	})
	if err != nil {
		return err
	}

	out, err := cmd.PersistentFlags().GetString("out")
	if err != nil {
		return err
	}
	out = utils.ExpandPath(out)
	var fi *os.File

	if fi, err = os.Create(out); err != nil && !os.IsExist(err) {
		return err
	}

	defer func(fi *os.File) {
		_ = fi.Close()
	}(fi)

	pool := workpool.New(10)
	perPage := 100
	totalPages := res.GetTotal() / perPage

	bar := pb.StartNew(res.GetTotal())
	var mx sync.Mutex

	for i := 0; i <= totalPages; i++ {
		ii := i

		pool.Do(func() error {
			defer bar.Increment()

			if err := client.Wait(cmd.Context()); err != nil {
				return err
			}

			res, _, err := client.Search.Repositories(cmd.Context(), query, &github.SearchOptions{
				TextMatch: false,
				ListOptions: github.ListOptions{
					Page:    ii,
					PerPage: perPage,
				},
			})
			if err != nil {
				return err
			}
			mx.Lock()
			defer mx.Unlock()

			for _, repository := range res.Repositories {
				if _, err = fmt.Fprintf(
					fi,
					"%s;%s;%d\n",
					repository.GetFullName(),
					repository.GetSSHURL(),
					repository.GetSize(),
				); err != nil {
					return err
				}
			}

			return nil
		})
	}

	if err := pool.Wait(); err != nil {
		return err
	}

	bar.Finish()

	return nil
}
