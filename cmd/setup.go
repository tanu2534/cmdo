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

	switch runtime.GOOS {
	case "windows":
		if strings.Contains(exepath, "cmd.exe") {
			return shellInfo{
				Name:       "CMD",
				ExePath:    exepath,
				ConfigPath: "",
				Type:       "cmd",
			}
		} else if strings.Contains(exepath, "powershell.exe") || strings.Contains(exepath, "pwsh.exe") {
			// Get profile path specific to THIS PowerShell executable
			profilePath := getPowerShellProfileForExe(shell)
			shellName := "PowerShell"
			if strings.Contains(exepath, "pwsh.exe") {
				shellName = "PowerShell Core"
			} else {
				shellName = "Windows PowerShell"
			}
			
			return shellInfo{
				Name:       shellName,
				ExePath:    exepath,
				ConfigPath: profilePath,
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
	case "darwin":
		// macOS shells
		if strings.Contains(exepath, "zsh") {
			return shellInfo{
				Name:       "Zsh",
				ExePath:    exepath,
				ConfigPath: filepath.Join(os.Getenv("HOME"), ".zshrc"),
				Type:       "zsh",
			}
		} else if strings.Contains(exepath, "bash") {
			return shellInfo{
				Name:       "Bash",
				ExePath:    exepath,
				ConfigPath: filepath.Join(os.Getenv("HOME"), ".bashrc"),
				Type:       "bash",
			}
		} else if strings.Contains(exepath, "fish") {
			return shellInfo{
				Name:       "Fish",
				ExePath:    exepath,
				ConfigPath: filepath.Join(os.Getenv("HOME"), ".config", "fish", "config.fish"),
				Type:       "fish",
			}
		}
	}

	return shellInfo{}
}

func getPowerShellProfileForExe(psExePath string) string {
	// Run the specific PowerShell executable to get ITS profile path
	cmd := exec.Command(psExePath, "-NoProfile", "-Command", "echo $PROFILE")
	output, err := cmd.Output()
	if err == nil {
		profilePath := strings.TrimSpace(string(output))
		if profilePath != "" {
			return profilePath
		}
	}

	// Fallback: determine based on executable name
	exepath := strings.ToLower(psExePath)
	userProfile := os.Getenv("USERPROFILE")
	
	if strings.Contains(exepath, "pwsh.exe") {
		// PowerShell Core (7+)
		return filepath.Join(userProfile, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	} else {
		// Windows PowerShell (5.1)
		return filepath.Join(userProfile, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	}
}

func getPowerShellProfile() string {
	userProfile := os.Getenv("USERPROFILE")
	psCorePath := filepath.Join(userProfile, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	winPSPath := filepath.Join(userProfile, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")

	if _, err := os.Stat(psCorePath); err == nil {
		return psCorePath
	}
	if _, err := os.Stat(winPSPath); err == nil {
		return winPSPath
	}
	return psCorePath
}

func getPowerShellHook(cmdoBinaryPath string) string {
	// Escape backslashes in the path for PowerShell
	escapedPath := strings.ReplaceAll(cmdoBinaryPath, `\`, `\\`)

	return fmt.Sprintf(`
# CMDO Command Logger Hook - DEBUG VERSION
$Global:__CmdoLastHistoryCount = 0
$Global:__CmdoInitialized = $false
$Global:__CmdoDebugLog = "$env:USERPROFILE\.cmdo\debug.log"

# Debug logging function
function Write-CmdoDebug {
	param($message)
	$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
	"[$timestamp] $message" | Out-File -Append -FilePath $Global:__CmdoDebugLog -ErrorAction SilentlyContinue
}

Write-CmdoDebug "=== CMDO Hook Loaded ==="

# Save original prompt if it exists
if (Test-Path Function:\prompt) {
	$Global:__CmdoOriginalPrompt = ${function:prompt}
	Write-CmdoDebug "Original prompt saved"
}

function Global:prompt {
	# Capture the exit code FIRST before any other commands
	$cmdoExitCode = $LASTEXITCODE
	if ($null -eq $cmdoExitCode) { $cmdoExitCode = 0 }
	
	# Get current history count
	$currentHistoryCount = (Get-History -ErrorAction SilentlyContinue | Measure-Object).Count
	
	Write-CmdoDebug "Prompt called - HistoryCount: $currentHistoryCount, LastCount: $Global:__CmdoLastHistoryCount, ExitCode: $cmdoExitCode"
	
	# Skip logging on first prompt (initialization)
	if (-not $Global:__CmdoInitialized) {
		$Global:__CmdoInitialized = $true
		$Global:__CmdoLastHistoryCount = $currentHistoryCount
		Write-CmdoDebug "Initialized - skipping first prompt"
	}
	# Log if a new command was executed
	elseif ($currentHistoryCount -gt $Global:__CmdoLastHistoryCount) {
		$lastEntry = Get-History -Count 1 -ErrorAction SilentlyContinue
		if ($lastEntry) {
			$cmdoCommand = $lastEntry.CommandLine
			$cmdoDir = $PWD.Path
			
			Write-CmdoDebug "Logging command: $cmdoCommand (exit: $cmdoExitCode, dir: $cmdoDir)"
			
			# Call cmdo log directly (synchronous for debugging)
			try {
				& "%s" log --command "$cmdoCommand" --exit-code $cmdoExitCode --pwd "$cmdoDir" 2>&1 | Out-File -Append -FilePath $Global:__CmdoDebugLog
				Write-CmdoDebug "Log command completed"
			} catch {
				Write-CmdoDebug "ERROR: $($_.Exception.Message)"
			}
		} else {
			Write-CmdoDebug "No history entry found"
		}
		$Global:__CmdoLastHistoryCount = $currentHistoryCount
	} else {
		Write-CmdoDebug "No new commands to log"
	}
	
	# Restore LASTEXITCODE for user scripts
	$global:LASTEXITCODE = $cmdoExitCode

	# Call original prompt if it exists
	if ($Global:__CmdoOriginalPrompt) {
		& $Global:__CmdoOriginalPrompt
	} else {
		"PS $($executionContext.SessionState.Path.CurrentLocation)$('>' * ($nestedPromptLevel + 1)) "
	}
}

# CMDO Autosuggestion Hook
if (Get-Module -ListAvailable -Name PSReadLine) {
	Set-PSReadLineKeyHandler -Key Tab -ScriptBlock {
		$line = $null
		$cursor = $null
		[Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
		if ($line.Length -ge 3) {
			$dir = $PWD.Path
			$suggestions = & "%s" completions --query "$line" --dir "$dir" 2>$null
			if ($suggestions) {
				$selected = $suggestions | Out-GridView -Title "CMDO Suggestions" -OutputMode Single
				if ($selected) {
					[Microsoft.PowerShell.PSConsoleReadLine]::RevertLine()
					[Microsoft.PowerShell.PSConsoleReadLine]::Insert($selected)
				}
			}
		} else {
			[Microsoft.PowerShell.PSConsoleReadLine]::TabCompleteNext()
		}
	}
}
`, escapedPath, escapedPath)
}

func getBashHook(cmdoBinaryPath string) string {
	return fmt.Sprintf(`
# CMDO Command Logger Hook
__cmdo_last_exit_code=0
__cmdo_last_history_id=""
__cmdo_initialized=0

function __cmdo_capture_exit() {
	__cmdo_last_exit_code=$?
}

function __cmdo_log() {
	local history_line=$(history 1)
	local history_id=$(echo "$history_line" | awk '{print $1}')
	local last_command=$(echo "$history_line" | sed 's/^[ ]*[0-9]*[ ]*//')
	local exit_code=$__cmdo_last_exit_code
	local current_dir=$(pwd)

	if [ "$__cmdo_initialized" -eq 0 ]; then
		__cmdo_initialized=1
		__cmdo_last_history_id="$history_id"
		return
	fi

	if [ -n "$last_command" ] && [ "$history_id" != "$__cmdo_last_history_id" ]; then
		__cmdo_last_history_id="$history_id"
		"%s" log --command "$last_command" --exit-code $exit_code --pwd "$current_dir" 2>/dev/null
	fi
}

if [[ ! "$PROMPT_COMMAND" =~ "__cmdo_capture_exit" ]]; then
	PROMPT_COMMAND="__cmdo_capture_exit; __cmdo_log${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
fi

# CMDO Autosuggestion
function __cmdo_suggest() {
	local query="${READLINE_LINE}"
	if [ "${#query}" -lt 3 ]; then
		return
	fi
	local selected
	selected=$("%s" completions --query "$query" --dir "$(pwd)" 2>/dev/null | fzf --height 40%% --reverse --prompt="CMDO> " --query="$query")
	if [ -n "$selected" ]; then
		READLINE_LINE="$selected"
		READLINE_POINT=${#READLINE_LINE}
	fi
}
bind -x '"\t":__cmdo_suggest'
`, cmdoBinaryPath, cmdoBinaryPath)
}

func getZshHook(cmdoBinaryPath string) string {
	return fmt.Sprintf(`
# CMDO Command Logger Hook
__cmdo_last_exit_code=0
__cmdo_initialized=0

function __cmdo_preexec() {
	__cmdo_last_command="$1"
}

function __cmdo_precmd() {
	__cmdo_last_exit_code=$?
	local current_dir=$(pwd)

	if [ "$__cmdo_initialized" -eq 0 ]; then
		__cmdo_initialized=1
		return
	fi

	if [ -n "$__cmdo_last_command" ]; then
		"%s" log --command "$__cmdo_last_command" --exit-code $__cmdo_last_exit_code --pwd "$current_dir" 2>/dev/null
		__cmdo_last_command=""
	fi
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec __cmdo_preexec
add-zsh-hook precmd __cmdo_precmd

# CMDO Autosuggestion
function __cmdo_suggest() {
	local query="$BUFFER"
	if [ "${#query}" -lt 3 ]; then
		zle expand-or-complete
		return
	fi
	local selected
	selected=$("%s" completions --query "$query" --dir "$(pwd)" 2>/dev/null | fzf --height 40%% --reverse --prompt="CMDO> " --query="$query")
	if [ -n "$selected" ]; then
		BUFFER="$selected"
		CURSOR=${#BUFFER}
		zle redisplay
	fi
}
zle -N __cmdo_suggest
bindkey "\t" __cmdo_suggest
`, cmdoBinaryPath, cmdoBinaryPath)
}

func getFishHook(cmdoBinaryPath string) string {
	return fmt.Sprintf(`
# CMDO Command Logger Hook
function __cmdo_log --on-event fish_postexec
	set -l exit_code $status
	set -l current_dir (pwd)
	"%s" log --command "$argv[1]" --exit-code $exit_code --pwd "$current_dir" 2>/dev/null
end

# CMDO Autosuggestion
function __cmdo_suggest
	set -l query (commandline)
	if test (string length -- $query) -lt 3
		return
	end
	set -l selected ("%s" completions --query "$query" --dir (pwd) 2>/dev/null | fzf --height 40%% --reverse --prompt="CMDO> " --query="$query")
	if test -n "$selected"
		commandline -- $selected
	end
end
bind \t __cmdo_suggest
`, cmdoBinaryPath, cmdoBinaryPath)
}

func addHookToConfigFile(shellInfo shellInfo, cmdoBinaryPath string) error {
	var hookScript string

	switch shellInfo.Type {
	case "powershell":
		hookScript = getPowerShellHook(cmdoBinaryPath)
	case "bash":
		hookScript = getBashHook(cmdoBinaryPath)
	case "zsh":
		hookScript = getZshHook(cmdoBinaryPath)
	case "fish":
		hookScript = getFishHook(cmdoBinaryPath)
	case "cmd":
		return setupCMDHook(cmdoBinaryPath)
	default:
		return fmt.Errorf("unsupported shell type")
	}

	if fileExists(shellInfo.ConfigPath) {
		content, err := os.ReadFile(shellInfo.ConfigPath)
		if err != nil {
			return err
		}
		if strings.Contains(string(content), "CMDO Command Logger Hook") {
			fmt.Printf("✓ Hook already exists in %s\n", shellInfo.ConfigPath)
			return nil
		}
	} else {
		configDir := filepath.Dir(shellInfo.ConfigPath)
		os.MkdirAll(configDir, 0755)
	}

	f, err := os.OpenFile(shellInfo.ConfigPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n" + hookScript + "\n")
	if err != nil {
		return err
	}

	fmt.Printf("✓ Hook added to %s\n", shellInfo.ConfigPath)
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func setupCMDHook(cmdoBinaryPath string) error {
	fmt.Println("⚠ Setting up CMD hook...")

	batchContent := fmt.Sprintf(`@echo off
setlocal enabledelayedexpansion
REM Capture the command
set "cmd=%%*"
REM Execute the command and capture exit code
%%*
set "exitcode=!errorlevel!"
REM Log to cmdo
"%s" log --command "!cmd!" --exit-code !exitcode! --pwd "%%CD%%" 2>nul
REM Return original exit code
exit /b !exitcode!
`, cmdoBinaryPath)

	cmdoWrapperPath := filepath.Join(filepath.Dir(cmdoBinaryPath), "cmdo-cmd-wrapper.bat")
	err := os.WriteFile(cmdoWrapperPath, []byte(batchContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to create wrapper: %v", err)
	}

	fmt.Println("\n⚠ CMD requires manual registry modification:")
	fmt.Println("  Run this command as Administrator in CMD:")
	fmt.Printf(`  reg add "HKCU\Software\Microsoft\Command Processor" /v AutoRun /t REG_SZ /d "@echo off" /f`)
	fmt.Println("\n\n  Or use DOSKEY (per-session):")
	fmt.Printf("  doskey cmdo=%s $*\n", cmdoBinaryPath)
	fmt.Println("\n  Note: CMD logging is limited. Consider using PowerShell or Git Bash for better experience.")

	return nil
}

func getInstalledBinaryPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	if strings.Contains(exePath, "go-build") {
		var possiblePaths []string

		switch runtime.GOOS {
		case "windows":
			possiblePaths = []string{
				filepath.Join(os.Getenv("GOPATH"), "bin", "cmdo.exe"),
				filepath.Join(os.Getenv("USERPROFILE"), "go", "bin", "cmdo.exe"),
				filepath.Join(os.Getenv("USERPROFILE"), "bin", "cmdo.exe"),
			}
		default:
			possiblePaths = []string{
				filepath.Join(os.Getenv("GOPATH"), "bin", "cmdo"),
				filepath.Join(os.Getenv("HOME"), "go", "bin", "cmdo"),
				filepath.Join(os.Getenv("HOME"), "bin", "cmdo"),
				"/usr/local/bin/cmdo",
			}
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}

		return "", fmt.Errorf("cmdo is not installed. Please run: go install")
	}

	return exePath, nil
}

func findMacOSShells() []string {
	var foundShells []string

	// Current user shell
	userShell := os.Getenv("SHELL")
	if userShell != "" {
		foundShells = append(foundShells, userShell)
		fmt.Printf("✓ Found current shell: %s\n", userShell)
	}

	// Common shell locations
	commonShells := []string{
		"/bin/zsh",
		"/bin/bash",
		"/usr/local/bin/zsh",
		"/usr/local/bin/bash",
		"/opt/homebrew/bin/zsh",
		"/opt/homebrew/bin/bash",
		"/usr/local/bin/fish",
		"/opt/homebrew/bin/fish",
	}

	for _, shell := range commonShells {
		if fileExists(shell) {
			// Avoid duplicates
			isDuplicate := false
			for _, existing := range foundShells {
				if existing == shell {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				foundShells = append(foundShells, shell)
				fmt.Printf("✓ Found shell: %s\n", shell)
			}
		}
	}

	return foundShells
}

var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{"setting"},
	Short:   "Setting up the CMDO logger server",
	Long:    "Use setup command in the binary for setting up the server locally.",
	Run: func(cmd *cobra.Command, args []string) {
		currentOS := runtime.GOOS

		cmdoBinaryPath, err := getInstalledBinaryPath()
		if err != nil {
			fmt.Printf(" Error: %v\n", err)
			fmt.Println("\n Installation steps:")
			switch currentOS {
			case "windows":
				fmt.Println("  1. Run: go build -o cmdo.exe")
				fmt.Printf("  2. Copy cmdo.exe to: %%USERPROFILE%%\\bin\\\n")
			default:
				fmt.Println("  1. Run: go build -o cmdo")
				fmt.Println("  2. Copy cmdo to: ~/bin/ or /usr/local/bin/")
			}
			fmt.Println("  3. Or run: go install")
			fmt.Println("  4. Then run: cmdo setup")
			return
		}

		fmt.Printf(" Using binary: %s\n\n", cmdoBinaryPath)

		var foundShells []string

		switch currentOS {
		case "windows":
			// Windows shell detection (existing code)
			gitBashPath := "C:\\Program Files\\Git\\bin\\bash.exe"
			if _, err := os.Stat(gitBashPath); err == nil {
				foundShells = append(foundShells, gitBashPath)
				fmt.Println("✓ Found Git Bash:", gitBashPath)
			}

			// PowerShell Core (pwsh) detection
			whereCmd := exec.Command("where", "pwsh")
			output, err := whereCmd.Output()
			if err == nil {
				paths := strings.Split(strings.TrimSpace(string(output)), "\r\n")
				for _, path := range paths {
					path = strings.TrimSpace(path)
					if path != "" {
						foundShells = append(foundShells, path)
						fmt.Println("✓ Found PowerShell Core:", path)
					}
				}
			}

			// Windows PowerShell detection
			wherePSCmd := exec.Command("where", "powershell")
			psOutput, psErr := wherePSCmd.Output()
			if psErr == nil {
				paths := strings.Split(strings.TrimSpace(string(psOutput)), "\r\n")
				for _, path := range paths {
					path = strings.TrimSpace(path)
					if path != "" {
						// Avoid duplicates
						isDuplicate := false
						for _, existing := range foundShells {
							if existing == path {
								isDuplicate = true
								break
							}
						}
						if !isDuplicate {
							foundShells = append(foundShells, path)
							fmt.Println("✓ Found Windows PowerShell:", path)
						}
					}
				}
			}

			// CMD detection
			cmdPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
			if _, err := os.Stat(cmdPath); err == nil {
				foundShells = append(foundShells, cmdPath)
				fmt.Println("✓ Found CMD:", cmdPath)
			}
		case "darwin":
			foundShells = findMacOSShells()
		}

		fmt.Printf("\n Total shells found: %d\n", len(foundShells))

		if len(foundShells) == 0 {
			fmt.Println(" No shells found to configure")
			return
		}

		fmt.Println("\n  Installing hooks...")
		successCount := 0

		for _, shellPath := range foundShells {
			shellInfo := identifyShell(shellPath)
			if shellInfo.Type == "" {
				continue
			}

			fmt.Printf("Setting up %s...\n", shellInfo.Name)
			err := addHookToConfigFile(shellInfo, cmdoBinaryPath)
			if err != nil {
				fmt.Printf("  Error: %v\n\n", err)
			} else {
				successCount++
			}
		}

		fmt.Printf("\nSetup complete! (%d/%d shells configured)\n", successCount, len(foundShells))
		fmt.Println("\n Next steps:")
		fmt.Println("  1. Restart your terminal, OR")
		switch currentOS {
		case "darwin":
			fmt.Println("  2. For Zsh: source ~/.zshrc")
			fmt.Println("  3. For Bash: source ~/.bashrc")
			fmt.Println("  4. For Fish: source ~/.config/fish/config.fish")
		default:
			fmt.Println("  2. For Bash: source ~/.bashrc")
			fmt.Println("  3. For PowerShell: . $PROFILE")
		}
		fmt.Println("\n Test it: Run any command and check 'cmdo serve'")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
