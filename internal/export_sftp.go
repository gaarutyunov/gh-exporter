package internal

import (
	"github.com/gaarutyunov/gh-exporter/pkg/sftpfs"
	"github.com/spf13/cobra"
)

func ExportSFTP(cmd *cobra.Command, args []string) error {
	fs, err := sftpfs.FromCmd(cmd, args)
	if err != nil {
		return err
	}

	return doExport(cmd.Context(), cmd.Parent(), args, fs)
}
