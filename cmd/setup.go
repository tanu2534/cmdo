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

type shellInfo struct {
	Name       string
	ExePath    string
	ConfigPath string
	Type       string
}

func identifyShell(shell string) shellInfo {
	exepath := strings.ToLower(shell)

	if strings.Contains(exepath, "cmd.exe") {
		return shellInfo{
			Name:       "CMD",
			ExePath:    exepath,
			ConfigPath: "",
			Type:       "cmd",
		}
	} else if strings.Contains(exepath, "powershell.exe") || strings.Contains(exepath, "pwsh.exe") {
		return shellInfo{
			Name:       "PowerShell",
			ExePath:    exepath,
			ConfigPath: getPowerShellProfile(),
			Type:       "powershell",
		}
	} else if strings.Contains(exepath, "bash.exe") {
		return shellInfo{
			Name:       "Git Bash",
			ExePath:    exepath,
			ConfigPath: filepath.Join(os.Getenv("USERPROFILE"), ".bashrc"),
			Type:       "bash",
		}
	}

	return shellInfo{}
}

func getPowerShellProfile() string {
	userProfile := os.Getenv("USERPROFILE")

	// PowerShell Core (pwsh)
	psCorePath := filepath.Join(userProfile, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")

	// Windows PowerShell (powershell)
	winPSPath := filepath.Join(userProfile, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")

	// Check which one exists, prefer Core
	if _, err := os.Stat(psCorePath); err == nil {
		return psCorePath
	}

	if _, err := os.Stat(winPSPath); err == nil {
		return winPSPath
	}

	// Return Core path (will be created if needed)
	return psCorePath
}

func getPowerShellHook(cmdoBinaryPath string) string {
	return fmt.Sprintf(`
# CMDO Command Logger Hook
$Global:__CmdoLastHistoryId = -1

function Invoke-CmdoLog {
    $history = Get-History -Count 1 -ErrorAction SilentlyContinue
    
    if ($history -and $history.Id -ne $Global:__CmdoLastHistoryId) {
        $Global:__CmdoLastHistoryId = $history.Id
        $lastCommand = $history.CommandLine
        $exitCode = $LASTEXITCODE
        if ($null -eq $exitCode) { $exitCode = 0 }
        $currentDir = $PWD.Path
        
        if ($lastCommand) {
            try {
                & '%s' log --command "$lastCommand" --exit-code $exitCode --pwd "$currentDir" 2>$null
            } catch {
                # Silently ignore logging errors
            }
        }
    }
}

# Save original prompt if exists
if (Test-Path Function:\prompt) {
    $Global:__CmdoOriginalPromptDef = ${function:prompt}.ToString()
}

function Global:prompt {
    Invoke-CmdoLog
    
    # Restore and execute original prompt
    if ($Global:__CmdoOriginalPromptDef) {
        $result = Invoke-Expression $Global:__CmdoOriginalPromptDef
        if ($result) { return $result }
    }
    
    # Default prompt if no original exists
    "PS $($executionContext.SessionState.Path.CurrentLocation)$('>' * ($nestedPromptLevel + 1)) "
}
`, cmdoBinaryPath)
}

func getBashHook(cmdoBinaryPath string) string {
	// Windows paths ko Git Bash compatible format me convert karo
	bashCompatiblePath := strings.ReplaceAll(cmdoBinaryPath, "\\", "/")

	return fmt.Sprintf(`
# CMDO Command Logger Hook
function __cmdo_log() {
    local last_command=$(history 1 | sed 's/^[ ]*[0-9]*[ ]*//')
    local exit_code=$?
    local current_dir=$(pwd)
    
    if [ -n "$last_command" ]; then
        "%s" log --command "$last_command" --exit-code $exit_code --pwd "$current_dir" 2>/dev/null
    fi
}

# Hook into PROMPT_COMMAND
if [[ ! "$PROMPT_COMMAND" =~ "__cmdo_log" ]]; then
    PROMPT_COMMAND="__cmdo_log${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
fi
`, bashCompatiblePath)
}

func addHookToConfigFile(shellInfo shellInfo, cmdoBinaryPath string) error {
	var hookScript string

	switch shellInfo.Type {
	case "powershell":
		hookScript = getPowerShellHook(cmdoBinaryPath)
	case "bash":
		hookScript = getBashHook(cmdoBinaryPath)
	case "cmd":
		return setupCMDHook(cmdoBinaryPath)
	default:
		return fmt.Errorf("unsupported shell type")
	}

	// Check if hook already exists
	if fileExists(shellInfo.ConfigPath) {
		content, err := os.ReadFile(shellInfo.ConfigPath)
		if err != nil {
			return err
		}

		if strings.Contains(string(content), "CMDO Command Logger Hook") {
			fmt.Printf("Hook already exists in %s\n", shellInfo.ConfigPath)
			return nil
		}
	} else {
		// Create config file directory if needed
		configDir := filepath.Dir(shellInfo.ConfigPath)
		os.MkdirAll(configDir, 0755)
	}

	// Append hook to config file
	f, err := os.OpenFile(shellInfo.ConfigPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n" + hookScript + "\n")
	if err != nil {
		return err
	}

	fmt.Printf("Hook added to %s\n", shellInfo.ConfigPath)
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func setupCMDHook(cmdoBinaryPath string) error {
	fmt.Println("CMD requires manual setup:")
	fmt.Println("Run this in CMD as Administrator:")
	fmt.Printf(`reg add "HKCU\Software\Microsoft\Command Processor" /v AutoRun /t REG_SZ /d "doskey cmdo=%s $*" /f`, cmdoBinaryPath)
	fmt.Println("\n(Automatic registry modification coming soon)")
	return nil
}

func getInstalledBinaryPath() (string, error) {
	// First try to get the current executable path
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Check if it's a temporary Go build path
	if strings.Contains(exePath, "go-build") {
		// Try to find installed binary in common locations
		possiblePaths := []string{
			filepath.Join(os.Getenv("GOPATH"), "bin", "cmdo.exe"),
			filepath.Join(os.Getenv("USERPROFILE"), "go", "bin", "cmdo.exe"),
			filepath.Join(os.Getenv("USERPROFILE"), "bin", "cmdo.exe"),
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}

		// If not found, suggest installation
		return "", fmt.Errorf("cmdo is not installed. Please run: go install")
	}

	return exePath, nil
}

var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{"setting"},
	Short:   "Setting up the CMDO logger server",
	Long:    "Use setup command in the binary for setting up the server locally.",
	Run: func(cmd *cobra.Command, args []string) {
		currentOS := runtime.GOOS

		if currentOS != "windows" {
			fmt.Println("Currently only Windows is supported")
			return
		}

		fmt.Println("🔍 Detecting installed shells...")

		// Get proper binary path
		cmdoBinaryPath, err := getInstalledBinaryPath()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println("\nInstallation steps:")
			fmt.Println("  1. Run: go build -o cmdo.exe")
			fmt.Printf("  2. Copy cmdo.exe to: %%USERPROFILE%%\\bin\\\n")
			fmt.Println("  3. Or run: go install")
			fmt.Println("  4. Then run: cmdo setup")
			return
		}

		fmt.Printf("✓ Using binary: %s\n\n", cmdoBinaryPath)

		userProfile := os.Getenv("USERPROFILE")
		var foundShells []string

		// Dotnet Global Tools
		dotnetPwsh := filepath.Join(userProfile, ".dotnet", "tools", "pwsh.exe")
		if _, err := os.Stat(dotnetPwsh); err == nil {
			foundShells = append(foundShells, dotnetPwsh)
			fmt.Println("Found Dotnet PowerShell:", dotnetPwsh)
		}

		// Scoop
		scoopPwsh := filepath.Join(userProfile, "scoop", "shims", "pwsh.exe")
		if _, err := os.Stat(scoopPwsh); err == nil {
			foundShells = append(foundShells, scoopPwsh)
			fmt.Println("Found Scoop PowerShell:", scoopPwsh)
		}

		// CMD
		cmdPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
		if _, err := os.Stat(cmdPath); err == nil {
			foundShells = append(foundShells, cmdPath)
			fmt.Println("Found CMD:", cmdPath)
		}

		// Use 'where' command to find PowerShell/pwsh
		if cmdPath != "" {
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

		// Git Bash
		gitBashPath := "C:\\Program Files\\Git\\bin\\bash.exe"
		if _, err := os.Stat(gitBashPath); err == nil {
			foundShells = append(foundShells, gitBashPath)
			fmt.Println("Found Git Bash:", gitBashPath)
		}

		// WSL check
		wslCmd := exec.Command("wsl.exe", "--list", "--quiet")
		output, err := wslCmd.Output()
		if err == nil {
			distros := strings.Split(string(output), "\n")
			for _, distro := range distros {
				distro = strings.TrimSpace(distro)
				if distro != "" {
					fmt.Println("Found WSL distro:", distro)
				}
			}
		}

		fmt.Printf("\n✓ Total shells found: %d\n", len(foundShells))

		if len(foundShells) == 0 {
			fmt.Println("No shells found to configure")
			return
		}

		fmt.Println("\nInstalling hooks...")

		for _, shellPath := range foundShells {
			shellInfo := identifyShell(shellPath)

			if shellInfo.Type == "" {
				continue
			}

			fmt.Printf("\n Setting up %s...\n", shellInfo.Name)

			err := addHookToConfigFile(shellInfo, cmdoBinaryPath)
			if err != nil {
				fmt.Printf("Error setting up %s: %v\n", shellInfo.Name, err)
			}
		}

		fmt.Println("\nSetup complete!")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Restart your terminal, OR")
		fmt.Println("  2. For Bash: source ~/.bashrc")
		fmt.Println("  3. For PowerShell: . $PROFILE")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
