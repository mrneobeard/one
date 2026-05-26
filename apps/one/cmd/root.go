package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "one",
	Short: "Monorepo management CLI",
	Long:  "one helps manage this monorepo with repeatable git and project workflows.",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
