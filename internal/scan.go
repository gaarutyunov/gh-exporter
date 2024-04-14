package internal

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/gaarutyunov/gh-exporter/pkg/gh"
	"github.com/gaarutyunov/gh-exporter/pkg/utils"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"regexp"
)

func Scan(cmd *cobra.Command, args []string) (err error) {
	in, err := cmd.PersistentFlags().GetString("in")
	if err != nil {
		return err
	}
	in = utils.ExpandPath(in)

	out, err := cmd.PersistentFlags().GetString("out")
	if err != nil {
		return err
	}
	out = utils.ExpandPath(out)

	format, err := cmd.PersistentFlags().GetString("format")
	if err != nil {
		return err
	}
	out = utils.ExpandPath(out)

	concurrency, err := cmd.PersistentFlags().GetInt("concurrency")
	if err != nil {
		return err
	}

	fIn, err := os.Open(in)
	if err != nil {
		return err
	}

	lines, err := utils.LineCounter(fIn)
	if err != nil {
		return err
	}
	_, err = fIn.Seek(0, 0)
	if err != nil {
		return err
	}

	bar := pb.StartNew(lines)

	fOut, err := os.OpenFile(out, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer func(fOut *os.File) {
		_ = fOut.Close()
	}(fOut)

	c := gh.NewClient(cmd.Context())

	linesCh := make(chan string, lines)
	errCh := make(chan error)
	done := make(chan struct{})

	defer func() {
		close(errCh)
		close(done)
	}()

	go func() {
		var wg errgroup.Group
		wg.SetLimit(concurrency)

		defer func() {
			_ = fIn.Close()
			if err != nil {
				errCh <- err
			}
		}()

		_ = utils.IterLines(fIn, func(line string) error {
			wg.Go(func() error {
				var url, sha string // TODO: switch to named args

				_, err := fmt.Sscanf(line, format, &url, &sha)
				if err != nil {
					return err
				}

				var owner, repo string

				r, err := regexp.Compile("https://github\\.com/([^/]+)/([^/]+).git")
				if err != nil {
					return err
				}
				for _, s := range r.FindAllStringSubmatch(url, 1) {
					owner, repo = s[1], s[2]
				}

				repository, resp, err := c.Repositories.Get(cmd.Context(), owner, repo)
				if resp.StatusCode == http.StatusNotFound {
					bar.AddTotal(-1)
					return nil
				}
				if err != nil {
					return err
				}

				linesCh <- fmt.Sprintf(
					"%s;%s;%d;%s\n",
					repository.GetFullName(),
					repository.GetSSHURL(),
					repository.GetSize(),
					sha,
				)

				return nil
			})

			return nil
		})

		if err := wg.Wait(); err != nil {
			errCh <- err
		}

		close(linesCh)
	}()

	go func() {
		for line := range linesCh {
			_, err := fOut.WriteString(line)
			if err != nil {
				errCh <- err
			}
			bar.Increment()
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
		return nil
	case err = <-errCh:
		close(linesCh)
		return
	}
}
