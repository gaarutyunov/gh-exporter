package main

import (
	"github.com/gaarutyunov/gh-exporter/internal"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"strings"
)

var (
	rootCmd = &cobra.Command{
		Use:   "gh_exporter",
		Short: "Explore and export GitHub repositories",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			verbosity, err := cmd.Root().PersistentFlags().GetString("verbosity")
			if err != nil {
				return err
			}

			level, err := logrus.ParseLevel(verbosity)
			if err != nil {
				return err
			}

			logrus.SetLevel(level)

			return nil
		},
	}

	searchCmd = &cobra.Command{
		Use:   "search",
		Short: "Explore repositories to be exported later",
		RunE:  internal.Search,
	}

	planCmd = &cobra.Command{
		Use:   "plan",
		Short: "Plan repositories export",
		Long:  "This command uses bin packing algorithm to group repositories for further optimized cloning",
		RunE:  internal.Plan,
	}

	exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Export planned repositories",
		RunE:  internal.Export,
	}

	scanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Scan repositories index from file",
		RunE:  internal.Scan,
	}
)

func init() {
	pFlags := rootCmd.PersistentFlags()
	levels := make([]string, 0, len(logrus.AllLevels))
	for _, level := range logrus.AllLevels {
		levels = append(levels, level.String())
	}
	pFlags.StringP("verbosity", "v", logrus.ErrorLevel.String(), "Verbosity level: "+strings.Join(levels, ", "))

	// search
	pFlags = searchCmd.PersistentFlags()
	pFlags.StringP("query", "q", "language:Python", "GitHub repos search query")
	pFlags.StringP("out", "o", "results.csv", "Search results file")
	pFlags.Int64P("limit", "l", -1, "Maximum number of repositories to export")
	pFlags.IntP("burst", "b", 1, "Rate limiter burst")

	// plan
	pFlags = planCmd.PersistentFlags()
	pFlags.Uint64P("capacity", "c", uint64(cache.GiByte), "Repository group capacity in bytes")
	pFlags.StringP("in", "i", "results.csv", "Search results input for planning")
	pFlags.StringP("out", "o", "plan.csv", "Plan file path")

	// export
	pFlags = exportCmd.PersistentFlags()
	pFlags.StringP("identity", "i", "~/.ssh/id_rsa", "SSH key path for cloning")
	pFlags.StringP("out", "o", "repos", "Output directory")
	pFlags.StringP("file", "f", "plan.csv", "Plan file path")
	pFlags.StringP("pattern", "p", "*.py", "Cloning file name pattern")
	pFlags.IntP("concurrency", "c", 10, "Cloning concurrency")
	pFlags.Bool("skip-remainder", false, "Skip exporting remainder")
	pFlags.Bool("only-remainder", false, "Export only remainder")
	pFlags.Bool("in-memory", false, "Use in-memory cloning")

	// scan
	pFlags = scanCmd.PersistentFlags()
	pFlags.IntP("concurrency", "c", 10, "Scanning concurrency")
	pFlags.StringP("in", "i", "input.spec", "Input file to scan")
	pFlags.StringP("out", "o", "results.csv", "Output file in search format")
	pFlags.StringP("format", "f", "%s %s", "Input file format")

	rootCmd.AddCommand(
		searchCmd,
		exportCmd,
		planCmd,
		scanCmd,
	)
}
