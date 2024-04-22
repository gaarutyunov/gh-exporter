package internal

import (
	"github.com/cheggaaa/pb/v3"
	"github.com/gaarutyunov/gh-exporter/pkg/gh"
	"github.com/gaarutyunov/gh-exporter/pkg/sftpfs"
	"github.com/gaarutyunov/gh-exporter/pkg/utils"
	"github.com/go-git/go-billy/v5/helper/chroot"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync/atomic"
)

func SyncSFTP(cmd *cobra.Command, args []string) (err error) {
	localDir, err := cmd.Parent().PersistentFlags().GetString("local")
	if err != nil {
		return err
	}
	localDir = utils.ExpandPath(localDir)

	localFs := osfs.New(localDir)

	remoteFs, err := sftpfs.FromCmd(cmd, args)
	if err != nil {
		return err
	}
	searchFile, err := cmd.Parent().PersistentFlags().GetString("search")
	if err != nil {
		return err
	}

	concurrency, err := cmd.Parent().PersistentFlags().GetInt("concurrency")
	if err != nil {
		return err
	}
	searchFile = utils.ExpandPath(searchFile)

	fIn, err := os.Open(searchFile)
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

	repoCh := make(chan *gh.Repo, lines)
	errCh := make(chan error)
	done := make(chan struct{})

	defer func() {
		close(errCh)
		close(done)
	}()

	go func() {
		defer func() {
			_ = fIn.Close()
			if err != nil {
				errCh <- err
			}
		}()

		_ = utils.IterLines(fIn, func(line string) error {
			vals := strings.Split(line, ";")
			fullName, sshUrl, size := vals[0], vals[1], vals[2]

			repo := gh.NewRepo(sshUrl, fullName, cast.ToUint64(size))
			if len(vals) > 3 {
				sha := strings.TrimSpace(vals[3])
				repo.SetSHA(sha)
			}

			repoCh <- repo

			return nil
		})

		close(repoCh)
	}()

	go func() {
		var wg errgroup.Group
		wg.SetLimit(concurrency)

		for repo := range repoCh {
			if _, err := localFs.Stat(repo.Dir()); os.IsNotExist(err) {
				bar.AddTotal(-1)
				continue
			}

			localRepoFs := chroot.New(localFs, repo.Dir())
			remoteRepoFs := chroot.New(remoteFs, repo.Dir())

			var counter atomic.Uint64

			err := util.Walk(localRepoFs, "", func(path string, info fs.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}
				counter.Add(1)

				wg.Go(func() error {
					defer func() {
						counter.Add(^uint64(0)) // decrement
						if counter.CompareAndSwap(0, 0) {
							bar.Increment()
						}
					}()

					src, err := localRepoFs.Open(path)
					if err != nil {
						return err
					}

					dst, err := remoteRepoFs.Create(path)
					if err != nil {
						return err
					}

					_, err = io.Copy(dst, src)
					if err != nil {
						return err
					}

					return nil
				})

				return nil
			})

			if err != nil {
				errCh <- err
			}
		}

		if err := wg.Wait(); err != nil {
			errCh <- err
		} else {
			done <- struct{}{}
		}
	}()

	select {
	case <-done:
		return nil
	case err = <-errCh:
		return err
	}
}
