package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "cmdo",
	Version: "1.0.7",
	Short:   "cmdo is a command logger tool",
	Long:    "cmdo is a command logger tool for tracking and managing command line history.",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Oops. An error while executing CMDO: '%s'\n", err)
		os.Exit(1)
	}
}
