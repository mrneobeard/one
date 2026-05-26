package cmd

import "github.com/spf13/cobra"

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage one project index data",
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}
