package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tanu2534/cmdo/database"
)



type Command struct {
	Text      string
	ExitCode  int
	Timestamp time.Time
	Folder    string
}

type Folder struct {
	Name     string
	Commands []Command
	Expanded bool
}

type model struct {
	allFolders []Folder   
	folders    []Folder
	folderIndex int
	search      textinput.Model
	status      string
	statusAt    time.Time
	width       int
}


var (
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63"))

	selectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("230"))

	mutedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
)



var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch CMDO terminal UI",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		dbPath := filepath.Join(home, ".cmdo", "cmdo.db")

		database.InitDB(dbPath)
		defer database.DB.Close()

		m := initialModel(database.DB)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if err := p.Start(); err != nil {
			fmt.Println("TUI error:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}


func (m *model) hasFolders() bool {
	return len(m.folders) > 0
}


func initialModel(db *sql.DB) model {
	ti := textinput.New()
	ti.Placeholder = "Search commands or folders"
	ti.Focus()
	ti.Width = 40

	folders := loadFolders(db)

	return model{
		allFolders: folders,
		folders:    folders,
		search:     ti,
	}
}



func loadFolders(db *sql.DB) []Folder {
	rows, err := db.Query(`
		SELECT command, exit_code, timestamp, directory
		FROM commands
		ORDER BY timestamp DESC
	`)
	if err != nil {
		return []Folder{}
	}
	defer rows.Close()

	folderMap := map[string][]Command{}

	for rows.Next() {
		var c Command
		var ts string

		if err := rows.Scan(&c.Text, &c.ExitCode, &ts, &c.Folder); err != nil {
			continue
		}

		c.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		folderMap[c.Folder] = append(folderMap[c.Folder], c)
	}

	var folders []Folder
	for folder, cmds := range folderMap {
		folders = append(folders, Folder{
			Name:     folder,
			Commands: cmds,
			Expanded: true,
		})
	}

	return folders
}



func filterFolders(all []Folder, query string) []Folder {
	if query == "" {
		return all
	}

	query = strings.ToLower(query)
	var result []Folder

	for _, folder := range all {

		// Folder name match → include whole folder
		if strings.Contains(strings.ToLower(folder.Name), query) {
			result = append(result, folder)
			continue
		}

		// Command match → include only matching commands
		var matched []Command
		for _, c := range folder.Commands {
			if strings.Contains(strings.ToLower(c.Text), query) {
				matched = append(matched, c)
			}
		}

		if len(matched) > 0 {
			result = append(result, Folder{
				Name:     folder.Name,
				Commands: matched,
				Expanded: true, 
			})
		}
	}

	return result
}



func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {

		case "q", "ctrl+c":
			return m, tea.Quit

		case "up":
			if m.hasFolders() && m.folderIndex > 0 {
				m.folderIndex--
			}

		case "down":
			if m.hasFolders() && m.folderIndex < len(m.folders)-1 {
				m.folderIndex++
			}

		case "enter":
			if m.hasFolders() {
				m.folders[m.folderIndex].Expanded =
					!m.folders[m.folderIndex].Expanded
			}

		case "c":
			if m.hasFolders() {
				m.status = "✔ Copied (mock)"
				m.statusAt = time.Now()
			}
		}
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)

	// 🔥 Apply global search
	m.folders = filterFolders(m.allFolders, m.search.Value())

	// Safe index reset
	if len(m.folders) == 0 {
		m.folderIndex = 0
	} else if m.folderIndex >= len(m.folders) {
		m.folderIndex = len(m.folders) - 1
	}

	return m, cmd
}

/* =======================
   View
======================= */

func (m model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render("CMDO ▸ Command History"))
	b.WriteString(mutedStyle.Render("    q Quit  ↑↓ Navigate  Enter Expand\n"))
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	// Search
	b.WriteString("Search: ")
	b.WriteString(m.search.View())
	b.WriteString("\n\n")

	// Empty state
	if len(m.folders) == 0 {
		b.WriteString(mutedStyle.Render("No matching commands or folders.\n"))
		b.WriteString(mutedStyle.Render("Try a different search.\n\n"))
		b.WriteString("q Quit\n")
		return b.String()
	}

	// Render folders
	for i, folder := range m.folders {
		cursor := "▶"
		if folder.Expanded {
			cursor = "▼"
		}

		line := fmt.Sprintf("%s %s (%d)", cursor, folder.Name, len(folder.Commands))
		if i == m.folderIndex {
			b.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}

		if folder.Expanded {
			for _, c := range folder.Commands {
				badge := successStyle.Render("✓")
				if c.ExitCode != 0 {
					badge = errorStyle.Render("✗")
				}

				b.WriteString(fmt.Sprintf(
					"   %-40s %s %s\n",
					c.Text,
					badge,
					c.Timestamp.Format("15:04 02 Jan"),
				))
			}
		}
	}

	// Footer
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	if time.Since(m.statusAt) < 3*time.Second {
		b.WriteString(successStyle.Render(m.status))
	}

	return b.String()
}
