package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var remotesSyncProjectRef string
var remotesSyncCreate bool

var gitRemotesSyncCmd = &cobra.Command{
	Use:   "sync [project]",
	Short: "Sync git remotes from one.yaml",
	Args:  cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateProjectFlagDirectory(remotesSyncProjectRef)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRef, _, err := projectRefFromInput(remotesSyncProjectRef, args)
		if err != nil {
			return err
		}

		remotes, ctx, err := loadProjectRemoteMap(projectRef)
		if err != nil {
			return err
		}

		if len(remotes) == 0 {
			fmt.Printf("No git remotes configured in %s\n", ctx.ProjectFile)
			return nil
		}

		for _, name := range sortedRemoteNames(remotes) {
			if err := addOrUpdateRemote(name, remotes[name], remotesSyncCreate); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	gitRemotesSyncCmd.Flags().StringVarP(&remotesSyncProjectRef, "project", "p", "", "Project directory (argument supports alias/name/path)")
	gitRemotesSyncCmd.Flags().BoolVar(&remotesSyncCreate, "create", false, "Create missing GitHub repositories with gh")
	gitRemotesCmd.AddCommand(gitRemotesSyncCmd)
}
