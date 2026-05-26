package cmd

import "github.com/spf13/cobra"

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git utilities driven by one.yaml",
}

func init() {
	rootCmd.AddCommand(gitCmd)
}
