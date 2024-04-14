package internal

import (
	"github.com/gaarutyunov/gh-exporter/pkg/sftpfs"
	"github.com/gaarutyunov/gh-exporter/pkg/utils"
	"github.com/pkg/sftp"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
)

func ExportSFTP(cmd *cobra.Command, args []string) error {
	user, err := cmd.PersistentFlags().GetString("user")
	if err != nil {
		return err
	}

	keyfile, err := cmd.PersistentFlags().GetString("keyfile")
	if err != nil {
		return err
	}
	keyfile = utils.ExpandPath(keyfile)
	file, err := os.ReadFile(keyfile)
	if err != nil {
		return err
	}

	passphrase, err := cmd.PersistentFlags().GetString("passphrase")
	if err != nil {
		return err
	}

	addr, err := cmd.PersistentFlags().GetString("addr")
	if err != nil {
		return err
	}

	signer, err := ssh.ParsePrivateKeyWithPassphrase(file, []byte(passphrase))
	if err != nil {
		return err
	}

	sshClient, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		User: user,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	})
	if err != nil {
		return err
	}

	client, err := sftp.NewClient(sshClient)
	if err != nil {
		return err
	}

	return doExport(cmd.Context(), cmd.Parent(), args, sftpfs.New(client))
}
