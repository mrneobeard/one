package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var remotesProjectRef string

var gitRemotesCmd = &cobra.Command{
	Use:   "remotes",
	Short: "Configure git remotes from one.yaml",
	Args:  cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateProjectFlagDirectory(remotesProjectRef)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRef, _, err := projectRefFromInput(remotesProjectRef, args)
		if err != nil {
			return err
		}

		cfg, ctx, err := loadProjectConfig(projectRef)
		if err != nil {
			return err
		}

		if len(cfg.Git.Remotes) == 0 {
			fmt.Printf("No git remotes configured in %s\n", ctx.ProjectFile)
			return nil
		}

		remoteNames := make([]string, 0, len(cfg.Git.Remotes))
		for name := range cfg.Git.Remotes {
			remoteNames = append(remoteNames, name)
		}
		sort.Strings(remoteNames)

		for _, name := range remoteNames {
			url := cfg.Git.Remotes[name]
			if err := addOrUpdateRemote(name, url); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	gitRemotesCmd.Flags().StringVarP(&remotesProjectRef, "project", "p", "", "Project directory (argument supports alias/name/path)")
	gitCmd.AddCommand(gitRemotesCmd)
}

func addOrUpdateRemote(name, url string) error {
	remoteURL, err := gitOutput("remote", "get-url", name)
	if err == nil {
		if remoteURL == url {
			fmt.Printf("Remote %s already configured\n", name)
			return nil
		}

		if err := gitRun("remote", "set-url", name, url); err != nil {
			return err
		}

		fmt.Printf("Updated remote %s -> %s\n", name, url)
		return nil
	}

	if !isUnknownRemoteErr(err) {
		return err
	}

	if err := gitRun("remote", "add", name, url); err != nil {
		return err
	}

	fmt.Printf("Added remote %s -> %s\n", name, url)
	return nil
}

func isUnknownRemoteErr(err error) bool {
	return strings.Contains(err.Error(), "No such remote")
}
