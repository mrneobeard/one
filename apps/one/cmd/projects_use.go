package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var useProjectRef string

var projectsUseCmd = &cobra.Command{
	Use:   "use",
	Short: "Set default project for one",
	Args:  cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateProjectFlagDirectory(useProjectRef)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRef, _, err := projectRefFromInput(useProjectRef, args)
		if err != nil {
			return err
		}

		repoRoot, err := getRepoRoot()
		if err != nil {
			return err
		}

		if projectRef == "" {
			return fmt.Errorf("project is required as an argument or --project")
		}

		idx, err := loadProjectsIndex(repoRoot)
		if err != nil {
			return err
		}

		projectDir, err := resolveExplicitProjectDir(projectRef, repoRoot, idx)
		if err != nil {
			return err
		}

		if _, err := parseProjectConfigFile(filepath.Join(projectDir, projectConfigFileName)); err != nil {
			return err
		}

		rel, err := pathFromRoot(repoRoot, projectDir)
		if err != nil {
			return err
		}

		if idx == nil {
			idx = &projectsIndex{Projects: []indexProject{}}
		}

		idx.DefaultProject = rel
		if err := writeProjectsIndex(repoRoot, idx); err != nil {
			return err
		}

		fmt.Printf("Default project set to %s\n", rel)
		return nil
	},
}

func init() {
	projectsUseCmd.Flags().StringVarP(&useProjectRef, "project", "p", "", "Project directory or alias")
	projectsCmd.AddCommand(projectsUseCmd)
}
