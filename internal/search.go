package internal

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/gaarutyunov/gh-exporter/pkg/gh"
	"github.com/gaarutyunov/gh-exporter/pkg/utils"
	"github.com/google/go-github/v45/github"
	"github.com/spf13/cobra"
	"os"
	"time"
)

const layout = "2006-01-02"

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

	if fi, err = utils.TryCreate(out); err != nil {
		return err
	}

	defer func(fi *os.File) {
		_ = fi.Close()
	}(fi)

	perPage := 100
	total := res.GetTotal()

	bar := pb.StartNew(res.GetTotal())

	now := time.Now()
	from := now.Add(-(time.Hour * 24 * 30))
	lastRange := fmt.Sprintf("%s..%s", from.Format(layout), now.Format(layout))

	for !bar.IsFinished() {
		if err := client.Wait(cmd.Context()); err != nil {
			return err
		}

		page := 0
		pageSize := perPage

		for pageSize >= perPage {
			res, _, err := client.Search.Repositories(cmd.Context(), query+" pushed:"+lastRange, &github.SearchOptions{
				Sort:      "updated",
				Order:     "desc",
				TextMatch: false,
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
			})
			if err != nil {
				return err
			}

			pageSize = len(res.Repositories)

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

				bar.Increment()
			}
			page++
		}

		lastFrom := from.Add(-(time.Hour * 24 * 1))
		from = lastFrom.Add(-(time.Hour * 24 * 30))
		lastRange = fmt.Sprintf("%s..%s", from.Format(layout), lastFrom.Format(layout))

		if bar.Current() >= int64(total) {
			bar.Finish()
		}
	}

	return nil
}
