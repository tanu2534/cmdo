package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanu2534/cmdo/database"
)

var logCmd = &cobra.Command{
	Use:     "log",
	Aliases: []string{"logs"},
	Short:   "Add log in the server",
	Long:    "log command adds the log in the server additional flags are --command, --exit-code, --pwd",
	// Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Printf("Addition of %s and %s = %s.\n\n", args[0], args[1], Add(args[0], args[1]))
		// fmt.Println(cmd)
		command, _ := cmd.Flags().GetString("command")
		exitCode, _ := cmd.Flags().GetString("exit-code")
		pwd, _ := cmd.Flags().GetString("pwd")

		if command != "" && pwd != "" {

			database.InitDB("./cmdo.db")

			fmt.Println(command, exitCode, pwd)
			database.InsertCmd(command, exitCode, pwd)

			defer database.DB.Close()
		}
	},
}

func init() {
	logCmd.Flags().String("command", "", "Command that was executed")
	logCmd.Flags().String("exit-code", "", "Exit code of the command")
	logCmd.Flags().String("pwd", "", "Working directory of the command")
	rootCmd.AddCommand(logCmd)
}
