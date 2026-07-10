// Command scpick is an interactive SCP/SFTP file transfer tool. See
// SPEC.md for the full design.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"scpick/internal/transfer"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "scpick: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "scpick",
		Short: "Interactive SCP/SFTP file transfer",
		Long: `scpick is a cross-platform interactive SCP/SFTP file transfer tool.

There are no path arguments: host, remote path, and local path are all
chosen through an interactive picker, one directory level at a time.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newPullCmd(), newPushCmd())
	return root
}

func newPullCmd() *cobra.Command {
	var recursive bool
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Download files from a remote host",
		Long: `Pull downloads files from a remote host to your local machine.

Steps: select a host (from ~/.ssh/config or entered manually) -> authenticate
-> browse and select remote file(s) -> browse and select a local destination
directory -> transfer.`,
		Example: "  scpick pull\n  scpick pull --recursive",
		RunE: func(cmd *cobra.Command, args []string) error {
			return transfer.RunPull(recursive)
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "recursively transfer directories, like scp -r")
	return cmd
}

func newPushCmd() *cobra.Command {
	var recursive bool
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Upload files to a remote host",
		Long: `Push uploads files from your local machine to a remote host.

Steps: select a host (from ~/.ssh/config or entered manually) -> authenticate
-> browse and select local file(s) -> browse and select a remote destination
directory -> transfer.`,
		Example: "  scpick push\n  scpick push --recursive",
		RunE: func(cmd *cobra.Command, args []string) error {
			return transfer.RunPush(recursive)
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "recursively transfer directories, like scp -r")
	return cmd
}
