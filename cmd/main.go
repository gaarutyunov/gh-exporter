package main

import (
	"context"
	"gh-exporter/internal"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var exportCmd = &cobra.Command{
	Use:  "gh_exporter",
	RunE: internal.Export,
}

func init() {
	pFlags := exportCmd.PersistentFlags()

	pFlags.StringP("query", "q", "q=language:python", "GItHub repos search query")
	pFlags.StringP("identity", "i", "~/.ssh/id_rsa", "SSH key path")
	pFlags.StringP("out", "o", "~/repos/python", "Output directory")
	pFlags.StringP("pattern", "p", "*.py", "Cloning file name pattern")
}

func main() {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	sigstop := make(chan os.Signal, 1)

	go func() {
		if err := exportCmd.ExecuteContext(ctx); err != nil {
			cancel()
			log.Fatalln(err)
		}

		sigstop <- syscall.Signal(0)
	}()

	signal.Notify(sigstop, syscall.SIGTERM, os.Interrupt)
	<-sigstop

	cancel()
}
