package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"runtime"
)

var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{"setting"},
	Short:   "Setting up the CMDO logger server",
	Long:    "Use setup command in the binary for setting up the server locally.",
	Run: func(cmd *cobra.Command, args []string) {
		currentOS := runtime.GOOS
		fmt.Println(currentOS)

		//2. Detect shell (bash/zsh/powershell/cmd)

		shell := os.Getenv("PROFILE")
		fmt.Println(shell)
		she := os.Getenv("SHELL")
		fmt.Println(she)

		ppid := os.Getppid()
		fmt.Println(ppid)

		hm, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting user home directory: %v\n", err)
		} else {
			fmt.Println(hm)
		}

		//3. Find shell config file path
		//   - bash: ~/.bashrc
		//   - zsh: ~/.zshrc
		//   - powershell: $PROFILE

		//4. Check if already installed

		//5. Add shell hook to config file
		//   Hook will call: cmdo log --command "..." --exit-code X --pwd "..."

		//6. Print success message + instructions
		//   "Restart terminal or run: source ~/.bashrc"
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
