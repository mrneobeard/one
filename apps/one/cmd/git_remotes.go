package cmd

import "github.com/spf13/cobra"

var gitRemotesCmd = &cobra.Command{
	Use:   "remotes",
	Short: "Manage git remotes from one.yaml",
}

func init() {
	gitCmd.AddCommand(gitRemotesCmd)
}
