package sftpfs

import (
	"github.com/gaarutyunov/gh-exporter/pkg/utils"
	"github.com/go-git/go-billy/v5"
	"github.com/pkg/sftp"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
)

func FromCmd(cmd *cobra.Command, args []string) (billy.Filesystem, error) {
	user, err := cmd.PersistentFlags().GetString("user")
	if err != nil {
		return nil, err
	}

	keyfile, err := cmd.PersistentFlags().GetString("keyfile")
	if err != nil {
		return nil, err
	}
	keyfile = utils.ExpandPath(keyfile)
	file, err := os.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}

	passphrase, err := cmd.PersistentFlags().GetString("passphrase")
	if err != nil {
		return nil, err
	}

	addr, err := cmd.PersistentFlags().GetString("addr")
	if err != nil {
		return nil, err
	}

	remote, err := cmd.PersistentFlags().GetString("remote")
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKeyWithPassphrase(file, []byte(passphrase))
	if err != nil {
		return nil, err
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
		return nil, err
	}

	client, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, err
	}

	return New(client, remote), nil
}
