package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var syncProjectRef string

var gitSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync configured subtree to remotes",
	Args:  cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateProjectFlagDirectory(syncProjectRef)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRef, _, err := projectRefFromInput(syncProjectRef, args)
		if err != nil {
			return err
		}

		cfg, ctx, err := loadProjectConfig(projectRef)
		if err != nil {
			return err
		}

		syncCfg := subtreeSyncConfig(cfg.Git)
		if !syncCfg.Subtree {
			fmt.Printf("Subtree sync disabled in %s\n", ctx.ProjectFile)
			return nil
		}

		if err := gitRun("push", "origin"); err != nil {
			return err
		}

		remotes := syncCfg.Remotes
		if len(remotes) == 0 {
			for name := range cfg.Git.Remotes {
				remotes = append(remotes, name)
			}
			sort.Strings(remotes)
		}

		for _, remote := range remotes {
			if err := gitRun("subtree", "push", "--prefix", ctx.ProjectRelPath, remote, "main"); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	gitSyncCmd.Flags().StringVarP(&syncProjectRef, "project", "p", "", "Project directory (argument supports alias/name/path)")
	gitCmd.AddCommand(gitSyncCmd)
}
