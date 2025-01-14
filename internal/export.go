package internal

import (
	"github.com/cheggaaa/pb/v3"
	"github.com/gaarutyunov/gh-exporter/gh"
	"github.com/gaarutyunov/gh-exporter/plan"
	"github.com/gaarutyunov/gh-exporter/utils"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func Export(cmd *cobra.Command, args []string) error {
	outDir, err := cmd.PersistentFlags().GetString("out")
	if err != nil {
		return err
	}
	outDir = utils.ExpandPath(outDir)

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

	pattern, err := cmd.PersistentFlags().GetString("pattern")
	if err != nil {
		return err
	}

	skipRemainder, err := cmd.PersistentFlags().GetBool("skip-remainder")
	if err != nil {
		return err
	}

	onlyRemainder, err := cmd.PersistentFlags().GetBool("only-remainder")
	if err != nil {
		return err
	}

	inMemory, err := cmd.PersistentFlags().GetBool("in-memory")
	if err != nil {
		return err
	}

	fin, err := plan.Open(planFile)
	if err != nil {
		return err
	}

	total := fin.Total(skipRemainder, onlyRemainder)

	bar := pb.StartNew(total)

	defer bar.Finish()

	outFs := osfs.New(outDir)

	ctx := cmd.Context()

	for group, isRemainder := range fin.Iter(skipRemainder, onlyRemainder) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var wg errgroup.Group
		wg.SetLimit(concurrency)

		group := group
		isRemainder := isRemainder

		for _, repoInfo := range group {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			repository := gh.NewRepo(repoInfo, nil)

			if ok, err := repository.Exists(outFs); err != nil {
				return err
			} else if ok {
				bar.AddTotal(-1)
				continue
			}

			wg.Go(func() error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				cloneFn := repository.CloneFS

				if inMemory && !isRemainder {
					cloneFn = repository.CloneMem
				}

				defer bar.Increment()
				if err := cloneFn(ctx, publicKey, pattern, outFs); err != nil {
					logrus.Errorf("error for %s: %s", repository.FullName(), err)
				}

				return nil
			})
		}

		if err = wg.Wait(); err != nil {
			return err
		}
	}

	return nil
}
