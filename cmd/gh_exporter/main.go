package main

import (
	"context"
	"github.com/gaarutyunov/gh-exporter/internal"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	rootCmd = &cobra.Command{
		Use:   "gh_exporter",
		Short: "Explore and export GitHub repositories",
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
)

func init() {
	// search
	pFlags := searchCmd.PersistentFlags()
	pFlags.StringP("query", "q", "q=language:python", "GItHub repos search query")
	pFlags.StringP("out", "o", "~/git-py/results.csv", "Search results file")

	// plan
	pFlags = planCmd.PersistentFlags()
	pFlags.Uint64P("capacity", "c", uint64(cache.GiByte), "Repository group capacity in bytes")
	pFlags.StringP("in", "i", "~/git-py/results.csv", "Search results input for planning")
	pFlags.StringP("out", "o", "~/git-py/plan.csv", "Plan file path")

	// export
	pFlags = exportCmd.PersistentFlags()
	pFlags.StringP("identity", "i", "~/.ssh/id_rsa", "SSH key path")
	pFlags.StringP("out", "o", "~/git-py/repos/python", "Output directory")
	pFlags.StringP("file", "f", "~/git-py/plan.csv", "Plan file path")
	pFlags.StringP("search", "s", "~/git-py/results.csv", "Search results file path to determine total")
	pFlags.StringP("pattern", "p", "*.py", "Cloning file name pattern")
	pFlags.IntP("concurrency", "c", 10, "Cloning concurrency")

	rootCmd.AddCommand(
		searchCmd,
		exportCmd,
		planCmd,
	)
}

func main() {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	sigstop := make(chan os.Signal, 1)

	go func() {
		if err := rootCmd.ExecuteContext(ctx); err != nil {
			cancel()
			log.Fatalln(err)
		}

		sigstop <- syscall.Signal(0)
	}()

	signal.Notify(sigstop, syscall.SIGTERM, os.Interrupt)
	<-sigstop

	cancel()
}
