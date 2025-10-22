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
			// Use global DB path
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

			// Insert the command
			database.InsertCmd(command, exitCode, pwd)

			// Auto-cleanup: Keep last 1000 commands OR 30 days (whichever is more)
			cleanupOldCommands(1000, 30)
		}
	},
}

// cleanupOldCommands removes old entries based on count and age limits
// Keeps last maxCount commands OR commands from last maxDays (whichever gives more commands)
func cleanupOldCommands(maxCount int, maxDays int) {
	// Strategy: Delete commands that are BOTH:
	// 1. Not in the most recent maxCount commands
	// 2. Older than maxDays

	// This ensures we keep at least maxCount recent commands
	// AND we keep all commands from the last maxDays even if > maxCount

	_, err := database.DB.Exec(`
		DELETE FROM commands 
		WHERE id NOT IN (
			SELECT id FROM commands 
			ORDER BY timestamp DESC 
			LIMIT ?
		) 
		AND timestamp < datetime('now', '-' || ? || ' days')
	`, maxCount, maxDays)

	if err != nil {
		// Silently fail - don't disrupt logging if cleanup fails
		// Can add logging here if needed for debugging
		return
	}
}

func init() {
	logCmd.Flags().String("command", "", "Command that was executed")
	logCmd.Flags().String("exit-code", "", "Exit code of the command")
	logCmd.Flags().String("pwd", "", "Working directory of the command")
	rootCmd.AddCommand(logCmd)
}
