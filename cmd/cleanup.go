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
	// Pattern to match the entire bash hook:
	// From "# CMDO Command Logger Hook" to the last "fi" that's part of the hook

	// Use regex to remove the hook section
	// This pattern matches:
	// 1. The comment line
	// 2. The function definition and body
	// 3. The PROMPT_COMMAND section with its if/fi block
	pattern := `(?s)# CMDO Command Logger Hook.*?^fi\s*$`

	re := regexp.MustCompile(pattern)
	cleaned := re.ReplaceAllString(content, "")

	// If regex didn't work, try line-by-line approach
	if strings.Contains(cleaned, "CMDO Command Logger Hook") {
		cleaned = removeBashHookLineByLine(content)
	}

	// Clean up excessive newlines
	cleaned = cleanupExcessiveNewlines(cleaned)

	return cleaned
}

func removeBashHookLineByLine(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inHook := false
	fiCount := 0

	for _, line := range lines {
		// Start of hook
		if strings.Contains(line, "# CMDO Command Logger Hook") {
			inHook = true
			fiCount = 0
			continue
		}

		if inHook {
			// Count 'fi' occurrences - bash hook has 2 'fi's
			if strings.TrimSpace(line) == "fi" {
				fiCount++
				// After second 'fi', hook ends
				if fiCount >= 2 {
					inHook = false
				}
				continue
			}
			// Skip all lines in hook
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func removePowerShellHook(content string) string {
	// Pattern to match PowerShell hook:
	// From "# CMDO Command Logger Hook" to the closing brace of "function Global:prompt"

	// Use regex to remove the hook section
	pattern := `(?s)# CMDO Command Logger Hook.*?^function Global:prompt \{.*?^}\s*$`

	re := regexp.MustCompile(pattern)
	cleaned := re.ReplaceAllString(content, "")

	// If regex didn't work, try line-by-line approach
	if strings.Contains(cleaned, "CMDO Command Logger Hook") {
		cleaned = removePowerShellHookLineByLine(content)
	}

	// Clean up excessive newlines
	cleaned = cleanupExcessiveNewlines(cleaned)

	return cleaned
}

func removePowerShellHookLineByLine(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inHook := false
	braceDepth := 0
	inPromptFunction := false

	for _, line := range lines {
		// Start of hook
		if strings.Contains(line, "# CMDO Command Logger Hook") {
			inHook = true
			braceDepth = 0
			inPromptFunction = false
			continue
		}

		if inHook {
			// Check if we're entering the Global:prompt function
			if strings.Contains(line, "function Global:prompt") {
				inPromptFunction = true
				braceDepth = 0
			}

			// Track braces
			if inPromptFunction {
				openBraces := strings.Count(line, "{")
				closeBraces := strings.Count(line, "}")
				braceDepth += openBraces - closeBraces

				// When we close the prompt function (braceDepth becomes 0 or negative)
				if closeBraces > 0 && braceDepth <= 0 {
					inHook = false
					inPromptFunction = false
					continue
				}
			}

			// Skip all lines in hook
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func cleanupExcessiveNewlines(content string) string {
	// Replace 3+ consecutive newlines with just 2
	re := regexp.MustCompile(`\n{3,}`)
	content = re.ReplaceAllString(content, "\n\n")

	// Ensure file ends with single newline
	content = strings.TrimRight(content, "\n") + "\n"

	return content
}

// func fileExists(path string) bool {
// 	_, err := os.Stat(path)
// 	return err == nil
// }

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

		fmt.Println("🧹 Cleaning up CMDO hooks...")

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
			fmt.Printf("🔍 Checking %s (%s)...\n", shellName, configPath)

			err := removeHookFromFile(configPath)
			if err != nil {
				if strings.Contains(err.Error(), "no CMDO hook found") {
					fmt.Printf("   ℹ️  No hook found\n")
					notFound++
				} else {
					fmt.Printf("   ❌ Error: %v\n", err)
					failed++
				}
			} else {
				fmt.Printf("   ✅ Hook removed successfully\n")
				removed++
			}
		}

		fmt.Printf("\n📊 Summary:\n")
		fmt.Printf("   ✅ Removed: %d\n", removed)
		fmt.Printf("   ℹ️  Not found: %d\n", notFound)
		if failed > 0 {
			fmt.Printf("   ❌ Failed: %d\n", failed)
		}

		if removed > 0 {
			fmt.Println("\n📝 Next steps:")
			fmt.Println("   1. Restart your terminal, OR")
			fmt.Println("   2. For Bash: source ~/.bashrc")
			fmt.Println("   3. For PowerShell: . $PROFILE")
		}

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
				fmt.Println("\n⚠️  WSL detected: Please manually remove hooks from WSL .bashrc if needed")
			}
		}

		fmt.Println("\n✨ Cleanup complete!")
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}
