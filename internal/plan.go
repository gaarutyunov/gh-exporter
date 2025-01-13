package internal

import (
	"bufio"
	"github.com/gaarutyunov/gh-exporter/binpack"
	"github.com/gaarutyunov/gh-exporter/gh"
	"github.com/gaarutyunov/gh-exporter/plan"
	"github.com/gaarutyunov/gh-exporter/utils"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/spf13/cobra"
)

func Plan(cmd *cobra.Command, args []string) (err error) {
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

	capacity, err := cmd.PersistentFlags().GetUint64("capacity")
	if err != nil {
		return err
	}

	capacity /= uint64(cache.KiByte)

	fin, err := utils.TryCreate(in)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := utils.TryCreate(out)
	if err != nil {
		return err
	}
	defer fout.Close()

	scanner := bufio.NewScanner(fin)

	var items []gh.RepoInfo

	for scanner.Scan() {
		repo, err := gh.RepoInfoFromString(scanner.Text())
		if err != nil {
			return err
		}
		items = append(items, repo)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	planFile := plan.New(binpack.FirstFit(items, capacity))

	_, err = fout.WriteString(planFile.String())
	if err != nil {
		return err
	}

	return nil
}
