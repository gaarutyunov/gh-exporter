package internal

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/gaarutyunov/gh-exporter/gh"
	"github.com/gaarutyunov/gh-exporter/utils"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"regexp"
)

var repoRegExp = regexp.MustCompile("https://github\\.com/([^/]+)/([^/]+).git")

func Scan(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()

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

	defer bar.Finish()

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

		for line := range utils.IterLines(fIn) {
			select {
			case <-ctx.Done():
				break
			default:
			}

			line := line

			wg.Go(func() error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				var url, sha string // TODO: switch to named args

				_, err := fmt.Sscanf(line, format, &url, &sha)
				if err != nil {
					return err
				}

				var owner, repo string

				for _, s := range repoRegExp.FindAllStringSubmatch(url, 1) {
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
		}

		if err := wg.Wait(); err != nil {
			errCh <- err
		}

		close(linesCh)
	}()

	go func() {
		for line := range linesCh {

			select {
			case <-ctx.Done():
				return
			default:
			}

			_, err := fOut.WriteString(line)
			if err != nil {
				errCh <- err
			}
			bar.Increment()
		}
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	case err = <-errCh:
		close(linesCh)
		return
	}
}
