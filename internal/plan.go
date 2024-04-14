package internal

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/gaarutyunov/gh-exporter/pkg/binpack"
	"github.com/gaarutyunov/gh-exporter/pkg/gh"
	"github.com/gaarutyunov/gh-exporter/pkg/utils"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var (
	ErrInvalidCast = errors.New("invalid cast")
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

	fin, err := utils.TryCreate(in)
	if err != nil {
		return err
	}
	defer func(fin *os.File) {
		err = fin.Close()
	}(fin)

	fout, err := utils.TryCreate(out)
	if err != nil {
		return err
	}
	defer func(fout *os.File) {
		err = fout.Close()
	}(fout)

	scanner := bufio.NewScanner(fin)

	var items []binpack.Packable

	for scanner.Scan() {
		vals := strings.Split(scanner.Text(), ";")
		fullName, sshUrl, size := vals[0], vals[1], vals[2]

		repo := gh.NewRepo(sshUrl, fullName, "", cast.ToUint64(size))
		if len(vals) > 3 {
			repo.SetSHA(vals[3])
		}
		items = append(items, repo)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	bins, remainder := binpack.FirstFit(items, capacity)

	for _, bin := range bins {
		for _, rr := range bin {
			repo, ok := rr.(*gh.Repo)
			if !ok {
				return ErrInvalidCast
			}

			if _, err = fmt.Fprintf(
				fout,
				"%s;%s;%d;%s\n",
				repo.FullName(),
				repo.SshURL(),
				repo.Size(),
				repo.SHA(),
			); err != nil {
				return err
			}
		}

		if _, err = fmt.Fprintln(fout); err != nil {
			return err
		}
	}

	if len(remainder) != 0 {
		if _, err = fmt.Fprintln(fout, "---"); err != nil {
			return err
		}

		for _, rr := range remainder {
			repo, ok := rr.(*gh.Repo)
			if !ok {
				return ErrInvalidCast
			}

			if _, err = fmt.Fprintf(
				fout,
				"%s;%s;%d;%s\n",
				repo.FullName(),
				repo.SshURL(),
				repo.Size(),
				repo.SHA(),
			); err != nil {
				return err
			}
		}
	}

	return
}
