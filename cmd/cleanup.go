package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func removeHookFromFile(filePath string) error {
	if !fileExists(filePath) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	originalContent := string(content)

	// Check if hook exists
	if !strings.Contains(originalContent, "CMDO Command Logger Hook") {
		return fmt.Errorf("no CMDO hook found")
	}

	var cleanedContent string

	// Detect file type
	if strings.HasSuffix(filePath, ".bashrc") || strings.HasSuffix(filePath, ".bash_profile") {
		cleanedContent = removeBashHook(originalContent)
	} else if strings.HasSuffix(filePath, ".ps1") {
		cleanedContent = removePowerShellHook(originalContent)
	} else {
		return fmt.Errorf("unknown file type: %s", filePath)
	}

	// Check if anything was actually removed
	if cleanedContent == originalContent {
		return fmt.Errorf("no CMDO hook found or failed to remove")
	}

	// Write cleaned content back
	err = os.WriteFile(filePath, []byte(cleanedContent), 0644)
	if err != nil {
		return err
	}

	return nil
}

func removeBashHook(content string) string {
	// Direct line-by-line approach works best for bash
	cleaned := removeBashHookLineByLine(content)

	// Clean up excessive newlines
	cleaned = cleanupExcessiveNewlines(cleaned)

	return cleaned
}

func removeBashHookLineByLine(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inHook := false
	skipNext := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// If we need to skip lines
		if skipNext > 0 {
			skipNext--
			continue
		}

		// Start of hook - look for the marker comment
		if strings.Contains(line, "# CMDO Command Logger Hook") {
			inHook = true

			// Find the end of the hook section
			// Look for the pattern: last "fi" that closes the PROMPT_COMMAND if block
			endIdx := findBashHookEnd(lines, i)
			if endIdx > i {
				// Skip all lines from current to end of hook
				skipNext = endIdx - i
				inHook = false
				continue
			}
		}

		// Only add line if not in hook
		if !inHook {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func findBashHookEnd(lines []string, startIdx int) int {
	// Look for the pattern that ends the hook
	// The hook ends after: "fi" that closes the PROMPT_COMMAND if statement

	fiCount := 0
	foundPromptCommand := false

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Track if we've seen PROMPT_COMMAND
		if strings.Contains(line, "PROMPT_COMMAND") {
			foundPromptCommand = true
		}

		// Count fi occurrences after we've seen the function definition
		if line == "fi" {
			fiCount++

			// The hook has only 1 fi now (for PROMPT_COMMAND if statement)
			// The __cmdo_log function doesn't have any if statements with fi
			if fiCount >= 1 && foundPromptCommand {
				return i
			}
		}
	}

	return startIdx
}

func removePowerShellHook(content string) string {
	// Direct line-by-line approach works best
	cleaned := removePowerShellHookLineByLine(content)

	// Clean up excessive newlines
	cleaned = cleanupExcessiveNewlines(cleaned)

	return cleaned
}

func removePowerShellHookLineByLine(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inHook := false
	skipNext := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// If we need to skip lines
		if skipNext > 0 {
			skipNext--
			continue
		}

		// Start of hook
		if strings.Contains(line, "# CMDO Command Logger Hook") {
			inHook = true

			// Find the end of the hook section
			endIdx := findPowerShellHookEnd(lines, i)
			if endIdx > i {
				// Skip all lines from current to end of hook
				skipNext = endIdx - i
				inHook = false
				continue
			}
		}

		// Only add line if not in hook
		if !inHook {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func findPowerShellHookEnd(lines []string, startIdx int) int {
	// The PowerShell hook ends after the closing brace of "function Global:prompt"

	braceDepth := 0
	inPromptFunction := false

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering the Global:prompt function
		if strings.Contains(line, "function Global:prompt") {
			inPromptFunction = true
			braceDepth = 0
		}

		if inPromptFunction {
			// Count opening and closing braces
			for _, char := range trimmedLine {
				if char == '{' {
					braceDepth++
				} else if char == '}' {
					braceDepth--

					// When braceDepth hits 0, we've closed the function
					if braceDepth == 0 {
						return i
					}
				}
			}
		}
	}

	return startIdx
}

func cleanupExcessiveNewlines(content string) string {
	// Replace 3+ consecutive newlines with just 2
	re := regexp.MustCompile(`\n{3,}`)
	content = re.ReplaceAllString(content, "\n\n")

	// Ensure file ends with single newline
	content = strings.TrimRight(content, "\n") + "\n"

	return content
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var cleanupCmd = &cobra.Command{
	Use:     "cleanup",
	Aliases: []string{"uninstall", "remove"},
	Short:   "Remove CMDO hooks from shell config files",
	Long:    "Removes all CMDO Command Logger hooks from detected shell configuration files",
	Run: func(cmd *cobra.Command, args []string) {
		currentOS := runtime.GOOS

		if currentOS != "windows" {
			fmt.Println("❌ Currently only Windows is supported")
			return
		}

		fmt.Println(" Cleaning up CMDO hooks...")

		userProfile := os.Getenv("USERPROFILE")
		configFiles := make(map[string]string)

		// Bash config
		bashrcPath := filepath.Join(userProfile, ".bashrc")
		if fileExists(bashrcPath) {
			configFiles["Git Bash"] = bashrcPath
		}

		// PowerShell Core profile
		psCorePath := filepath.Join(userProfile, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		if fileExists(psCorePath) {
			configFiles["PowerShell Core"] = psCorePath
		}

		// Windows PowerShell profile
		winPSPath := filepath.Join(userProfile, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		if fileExists(winPSPath) {
			configFiles["Windows PowerShell"] = winPSPath
		}

		if len(configFiles) == 0 {
			fmt.Println("ℹ️  No shell config files found")
			return
		}

		removed := 0
		notFound := 0
		failed := 0

		for shellName, configPath := range configFiles {
			fmt.Printf(" Checking %s (%s)...\n", shellName, configPath)

			err := removeHookFromFile(configPath)
			if err != nil {
				if strings.Contains(err.Error(), "no CMDO hook found") {
					fmt.Printf("  No hook found\n")
					notFound++
				} else {
					fmt.Printf("   ❌ Error: %v\n", err)
					failed++
				}
			} else {
				fmt.Printf("    Hook removed successfully\n")
				removed++
			}
		}

		fmt.Printf("\n Summary:\n")
		fmt.Printf("    Removed: %d\n", removed)
		fmt.Printf("  Not found: %d\n", notFound)
		if failed > 0 {
			fmt.Printf("   ❌ Failed: %d\n", failed)
		}

		// if removed > 0 {
		// 	fmt.Println("\nNext steps:")
		// 	fmt.Println("   1. Restart your terminal, OR")
		// 	fmt.Println("   2. For Bash: source ~/.bashrc")
		// 	fmt.Println("   3. For PowerShell: . $PROFILE")
		// }

		// Check for WSL
		wslCmd := exec.Command("wsl.exe", "--list", "--quiet")
		output, err := wslCmd.Output()
		if err == nil {
			distros := strings.Split(string(output), "\n")
			hasWSL := false
			for _, distro := range distros {
				distro = strings.TrimSpace(distro)
				if distro != "" {
					hasWSL = true
					break
				}
			}
			if hasWSL {
				fmt.Println("\nWSL detected: Please manually remove hooks from WSL .bashrc if needed")
			}
		}

		fmt.Println("\n Cleanup complete!")
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}
