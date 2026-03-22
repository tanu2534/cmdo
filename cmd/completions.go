package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanu2534/cmdo/database"
)

var completionsCmd = &cobra.Command{
	Use:    "completions",
	Short:  "Return matching commands for shell autosuggestion",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		query, _ := cmd.Flags().GetString("query")
		dir, _ := cmd.Flags().GetString("dir")

		if len(query) < 3 {
			return
		}

		home, _ := os.UserHomeDir()
		dbPath := filepath.Join(home, ".cmdo", "cmdo.db")
		database.InitDB(dbPath)
		defer database.DB.Close()

		results := queryCompletions(query, dir)
		for _, r := range results {
			fmt.Println(r)
		}
	},
}

func queryCompletions(query, dir string) []string {
	like := "%" + strings.ToLower(query) + "%"

	// First try current directory
	if dir != "" {
		rows, err := database.DB.Query(
			`SELECT DISTINCT command FROM commands
			 WHERE directory = ? AND LOWER(command) LIKE ?
			 ORDER BY timestamp DESC LIMIT 10`,
			dir, like,
		)
		if err == nil {
			defer rows.Close()
			var results []string
			for rows.Next() {
				var c string
				rows.Scan(&c)
				results = append(results, c)
			}
			if len(results) > 0 {
				return results
			}
		}
	}

	// Fallback: global search
	rows, err := database.DB.Query(
		`SELECT DISTINCT command FROM commands
		 WHERE LOWER(command) LIKE ?
		 ORDER BY timestamp DESC LIMIT 10`,
		like,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var c string
		rows.Scan(&c)
		results = append(results, c)
	}
	return results
}

func init() {
	completionsCmd.Flags().String("query", "", "Search query")
	completionsCmd.Flags().String("dir", "", "Current directory")
	rootCmd.AddCommand(completionsCmd)
}
