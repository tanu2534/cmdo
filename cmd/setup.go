package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
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
		if currentOS == "windows" {
			fmt.Println("Windows system mil gya h")

			userProfile := os.Getenv("USERPROFILE")
			// localAppData := os.Getenv("LOCALAPPDATA") // Ye correct hai
			// programFiles := os.Getenv("ProgramFiles")

			var foundShells []string

			// 1. PowerShell - Program Files (sabhi versions check karo)
			// psBasePath := filepath.Join(programFiles, "PowerShell")
			// if entries, err := os.ReadDir(psBasePath); err == nil {
			// 	for _, entry := range entries {
			// 		if entry.IsDir() {
			// 			pwshPath := filepath.Join(psBasePath, entry.Name(), "pwsh.exe")
			// 			if _, err := os.Stat(pwshPath); err == nil {
			// 				foundShells = append(foundShells, pwshPath)
			// 				fmt.Println("Found PowerShell:", pwshPath)
			// 			}
			// 		}
			// 	}
			// }

			// 2. PowerShell - Store Apps
			// windowsAppsPath := filepath.Join(localAppData, "Microsoft", "WindowsApps")
			// if entries, err := os.ReadDir(windowsAppsPath); err == nil {
			// 	for _, entry := range entries {
			// 		// Prefix match for both stable and preview
			// 		if strings.HasPrefix(entry.Name(), "Microsoft.PowerShell") {
			// 			pwshPath := filepath.Join(windowsAppsPath, entry.Name(), "pwsh.exe")
			// 			if _, err := os.Stat(pwshPath); err == nil {
			// 				foundShells = append(foundShells, pwshPath)
			// 				fmt.Println("Found Store PowerShell:", pwshPath)
			// 			}
			// 		}
			// 	}
			// }

			// 3. PowerShell - Dotnet Global Tools
			dotnetPwsh := filepath.Join(userProfile, ".dotnet", "tools", "pwsh.exe")
			if _, err := os.Stat(dotnetPwsh); err == nil {
				foundShells = append(foundShells, dotnetPwsh)
				fmt.Println("Found Dotnet PowerShell:", dotnetPwsh)
			}

			// 4. PowerShell - Scoop
			scoopPwsh := filepath.Join(userProfile, "scoop", "shims", "pwsh.exe")
			if _, err := os.Stat(scoopPwsh); err == nil {
				foundShells = append(foundShells, scoopPwsh)
				fmt.Println("Found Scoop PowerShell:", scoopPwsh)
			}

			// 5. CMD (hamesha hota hai)
			cmdPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
			if _, err := os.Stat(cmdPath); err == nil {
				foundShells = append(foundShells, cmdPath)
				fmt.Println("Found CMD:", cmdPath)
			}

			if cmdPath != "" {
				// PowerShell dhundo using 'where' command
				cmd := exec.Command("where", "powershell")
				output, err := cmd.Output()
				if err == nil {
					paths := strings.Split(strings.TrimSpace(string(output)), "\n")
					for _, path := range paths {
						path = strings.TrimSpace(path)
						if path != "" {
							fmt.Println("Found PowerShell via 'where':", path)
							foundShells = append(foundShells, path)
						}
					}
				}

				// pwsh bhi dhundo
				cmd = exec.Command("where", "pwsh")
				output, err = cmd.Output()
				if err == nil {
					paths := strings.Split(strings.TrimSpace(string(output)), "\n")
					for _, path := range paths {
						path = strings.TrimSpace(path)
						if path != "" {
							fmt.Println("Found pwsh via 'where':", path)
							foundShells = append(foundShells, path)
						}
					}
				}
			}
			// 6. Git Bash
			gitBashPath := "C:\\Program Files\\Git\\bin\\bash.exe"
			if _, err := os.Stat(gitBashPath); err == nil {
				foundShells = append(foundShells, gitBashPath)
				fmt.Println("Found Git Bash:", gitBashPath)
			}

			// 7. WSL check karo
			cmd := exec.Command("wsl.exe", "--list", "--quiet")
			output, err := cmd.Output()
			if err == nil {
				distros := strings.Split(string(output), "\n")
				for _, distro := range distros {
					distro = strings.TrimSpace(distro)
					if distro != "" {
						fmt.Println("Found WSL distro:", distro)
					}
				}
			}

			fmt.Printf("\nTotal shells found: %d\n", len(foundShells))
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
