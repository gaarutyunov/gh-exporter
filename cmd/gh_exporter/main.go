package main

import (
	"context"
	"github.com/gaarutyunov/gh-exporter/internal"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

	exportSFTPCmd = &cobra.Command{
		Use:   "sftp",
		Short: "Export planned repositories to SFTP target",
		RunE:  internal.ExportSFTP,
	}

	scanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Scan repositories index from file",
		RunE:  internal.Scan,
	}

	syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Sync repositories",
	}

	syncSFTPCmd = &cobra.Command{
		Use:   "sftp",
		Short: "Sync repositories of SFTP target",
		RunE:  internal.SyncSFTP,
	}
)

func init() {
	// search
	pFlags := searchCmd.PersistentFlags()
	pFlags.StringP("query", "q", "q=language:python", "GitHub repos search query")
	pFlags.StringP("out", "o", "~/git-py/results.csv", "Search results file")

	// plan
	pFlags = planCmd.PersistentFlags()
	pFlags.Uint64P("capacity", "c", uint64(cache.GiByte), "Repository group capacity in bytes")
	pFlags.StringP("in", "i", "~/git-py/results.csv", "Search results input for planning")
	pFlags.StringP("out", "o", "~/git-py/plan.csv", "Plan file path")

	// export
	pFlags = exportCmd.PersistentFlags()
	pFlags.StringP("identity", "i", "~/.ssh/id_rsa", "SSH key path for cloning")
	pFlags.StringP("out", "o", "~/git-py/repos/python", "Output directory")
	pFlags.StringP("file", "f", "~/git-py/plan.csv", "Plan file path")
	pFlags.StringP("search", "s", "~/git-py/results.csv", "Search results file path to determine total")
	pFlags.StringP("pattern", "p", "*.py", "Cloning file name pattern")
	pFlags.IntP("concurrency", "c", 10, "Cloning concurrency")

	// export sftp
	pFlags = exportSFTPCmd.PersistentFlags()
	pFlags.StringP("addr", "A", "cluster.hpc.hse.ru:2222", "SFTP target host and port")
	pFlags.StringP("keyfile", "K", "~/.ssh/id_hpc.pem", "SSH private key file path")
	pFlags.StringP("passphrase", "P", "", "Passphrase for keyfile")
	pFlags.StringP("user", "U", "gaarutyunov", "SSH user")
	pFlags.StringP("remote", "R", "~/git-py/repos/python", "Remote output directory")

	exportCmd.AddCommand(
		exportSFTPCmd,
	)

	// scan
	pFlags = scanCmd.PersistentFlags()
	pFlags.IntP("concurrency", "c", 10, "Scanning concurrency")
	pFlags.StringP("in", "i", "input.spec", "Input file to scan")
	pFlags.StringP("out", "o", "~/git-py/results.csv", "Output file in search format")
	pFlags.StringP("format", "f", "%s %s", "Input file format")

	// sync
	pFlags = syncCmd.PersistentFlags()
	pFlags.StringP("identity", "i", "~/.ssh/id_rsa", "SSH key path for cloning")
	pFlags.StringP("local", "l", "~/git-py/repos/python", "Output directory")
	pFlags.StringP("search", "s", "~/git-py/results.csv", "Search results file path to determine total")
	pFlags.StringP("pattern", "p", "*.py", "Cloning file name pattern")
	pFlags.IntP("concurrency", "c", 10, "Cloning concurrency")

	// sync sftp
	pFlags = syncSFTPCmd.PersistentFlags()
	pFlags.StringP("addr", "A", "cluster.hpc.hse.ru:2222", "SFTP target host and port")
	pFlags.StringP("keyfile", "K", "~/.ssh/id_hpc.pem", "SSH private key file path")
	pFlags.StringP("passphrase", "P", "", "Passphrase for keyfile")
	pFlags.StringP("user", "U", "gaarutyunov", "SSH user")
	pFlags.StringP("remote", "R", "~/git-py/repos/python", "Remote output directory")

	syncCmd.AddCommand(
		syncSFTPCmd,
	)

	rootCmd.AddCommand(
		searchCmd,
		exportCmd,
		planCmd,
		scanCmd,
		syncCmd,
	)
}

func main() {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	sigstop := make(chan os.Signal, 1)

	go func() {
		if err := rootCmd.ExecuteContext(ctx); err != nil {
			cancel()
			logrus.Fatalln(err)
		}

		sigstop <- syscall.Signal(0)
	}()

	signal.Notify(sigstop, syscall.SIGTERM, os.Interrupt)
	<-sigstop

	cancel()
}
