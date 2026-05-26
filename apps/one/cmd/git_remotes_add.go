package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var remotesAddCreate bool

var gitRemotesAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add or update a single git remote",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		url := args[1]

		if err := addOrUpdateRemote(name, url, remotesAddCreate); err != nil {
			return err
		}

		fmt.Printf("Remote %s synced\n", name)
		return nil
	},
}

func init() {
	gitRemotesAddCmd.Flags().BoolVar(&remotesAddCreate, "create", false, "Create missing GitHub repository with gh")
	gitRemotesCmd.AddCommand(gitRemotesAddCmd)
}
