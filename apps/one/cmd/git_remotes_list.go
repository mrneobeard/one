package cmd

import (
	"github.com/spf13/cobra"
)

var gitRemotesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured git remotes",
	RunE: func(cmd *cobra.Command, args []string) error {
		remotes, err := listGitRemotes()
		if err != nil {
			return err
		}

		for _, remote := range remotes {
			url, urlErr := gitOutput("remote", "get-url", remote)
			if urlErr != nil {
				return urlErr
			}

			cmd.Printf("%s %s\n", remote, url)
		}

		return nil
	},
}

func init() {
	gitRemotesCmd.AddCommand(gitRemotesListCmd)
}
