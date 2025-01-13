package internal

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/gaarutyunov/gh-exporter/gh"
	"github.com/gaarutyunov/gh-exporter/utils"
	"github.com/google/go-github/v45/github"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"sync/atomic"
)

const reposPerPage = 100

func Search(cmd *cobra.Command, args []string) error {
	query, err := cmd.PersistentFlags().GetString("query")
	if err != nil {
		return err
	}

	limit, err := cmd.PersistentFlags().GetInt64("limit")
	if err != nil {
		return err
	}

	burst, err := cmd.PersistentFlags().GetInt("burst")
	if err != nil {
		return err
	}

	client := gh.NewClient(cmd.Context())

	searchLimiter := gh.NewLimiter(
		client,
		gh.WithLimit(gh.SearchLimit),
		gh.WithBurst(burst),
	)

	coreLimiter := gh.NewLimiter(
		client,
		gh.WithLimit(gh.CoreLimit),
		gh.WithBurst(burst),
	)

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

	defer fi.Close()

	var counter, doneCounter atomic.Int64

	if limit > 0 {
		limit = min(limit, int64(res.GetTotal()))

		counter.Store(limit)
		doneCounter.Store(limit)
	} else {
		counter.Store(int64(res.GetTotal()))
		doneCounter.Store(int64(res.GetTotal()))
	}

	bar := pb.StartNew(int(limit))

	defer bar.Finish()

	perPage := min(int(limit), reposPerPage)

	page := 0

	repoCh := make(chan *gh.Repo)

	defer close(repoCh)

	go func() {
		for repo := range repoCh {
			select {
			case <-cmd.Context().Done():
				return
			default:
			}

			err := coreLimiter.Wait(cmd.Context())
			if err != nil {
				continue
			}

			defaultBranch := repo.GetDefaultBranch()

			commits, _, err := client.Repositories.ListCommits(cmd.Context(), repo.Owner(), repo.Name(), &github.CommitsListOptions{
				SHA: defaultBranch,
				ListOptions: github.ListOptions{
					Page:    0,
					PerPage: 1,
				},
			})
			if err != nil {
				logrus.Errorf("List commits for %s err: %v", repo.FullName(), err)
				bar.AddTotal(-1)
				doneCounter.Add(-1)
				continue
			}

			if len(commits) > 0 {
				repo.SetSHA(commits[0].GetSHA())
			}

			if _, err = fmt.Fprintln(fi, repo); err != nil {
				bar.AddTotal(-1)
				doneCounter.Add(-1)
				continue
			}

			bar.Increment()
			doneCounter.Add(-1)
		}
	}()

	for counter.Load() > 0 {
		select {
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		default:
		}

		if err := searchLimiter.Wait(cmd.Context()); err != nil {
			return err
		}

		res, _, err := client.Search.Repositories(cmd.Context(), query, &github.SearchOptions{
			TextMatch: false,
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return err
		}

		l := min(len(res.Repositories), int(counter.Load()))

		for i := 0; i < l; i++ {
			repository := res.Repositories[i]

			repoCh <- gh.NewRepo(
				gh.NewRepoInfo(
					repository.GetFullName(),
					repository.GetSSHURL(),
					uint64(repository.GetSize()),
				),
				repository,
			)

			counter.Add(-1)
		}

		page++
	}

	for !doneCounter.CompareAndSwap(0, 0) {
		select {
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		default:
		}
	}

	return nil
}
