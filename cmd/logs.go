package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tanu2534/cmdo/database"
)

var logCmd = &cobra.Command{
	Use:     "log",
	Aliases: []string{"logs"},
	Short:   "Add log in the server",
	Long:    "log command adds the log in the server additional flags are --command, --exit-code, --pwd",
	Run: func(cmd *cobra.Command, args []string) {
		command, _ := cmd.Flags().GetString("command")
		exitCode, _ := cmd.Flags().GetString("exit-code")
		pwd, _ := cmd.Flags().GetString("pwd")

		if command != "" && pwd != "" {
			// ✅ Use global DB path instead of ./cmdo.db
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Println("Error getting home directory:", err)
				return
			}

			// Create .cmdo directory if it doesn't exist
			cmdoDir := filepath.Join(homeDir, ".cmdo")
			os.MkdirAll(cmdoDir, 0755)

			dbPath := filepath.Join(cmdoDir, "cmdo.db")

			database.InitDB(dbPath)
			defer database.DB.Close()

			fmt.Println(command, exitCode, pwd)
			database.InsertCmd(command, exitCode, pwd)
		}
	},
}

func init() {
	logCmd.Flags().String("command", "", "Command that was executed")
	logCmd.Flags().String("exit-code", "", "Exit code of the command")
	logCmd.Flags().String("pwd", "", "Working directory of the command")
	rootCmd.AddCommand(logCmd)
}
