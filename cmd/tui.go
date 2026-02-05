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
	"github.com/charmbracelet/bubbles/viewport"
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
	search     textinput.Model
	viewport   viewport.Model
	width      int
	height     int
	status     string
	statusAt   time.Time
}



var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	selected    = lipgloss.NewStyle().Background(lipgloss.Color("237"))
	muted       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	ok          = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	bad         = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)



var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch CMDO terminal UI",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		dbPath := filepath.Join(home, ".cmdo", "cmdo.db")

		database.InitDB(dbPath)
		defer database.DB.Close()

		p := tea.NewProgram(initialModel(database.DB), tea.WithAltScreen())
		if err := p.Start(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}



func initialModel(db *sql.DB) model {
	ti := textinput.New()
	ti.Placeholder = "Search commands or folders"
	ti.Focus()
	ti.Width = 40

	vp := viewport.New(0, 0)

	folders := loadFolders(db)

	return model{
		allFolders: folders,
		folders:    folders,
		search:     ti,
		viewport:   vp,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}



func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 5
		footerHeight := 2

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - footerHeight

	case tea.KeyMsg:
		switch msg.String() {

		case "q", "ctrl+c":
			return m, tea.Quit

		case "up":
			m.viewport.LineUp(1)

		case "down":
			m.viewport.LineDown(1)

		case "pgup":
			m.viewport.HalfViewUp()

		case "pgdown":
			m.viewport.HalfViewDown()
		}
	}

	m.search, cmd = m.search.Update(msg)
	m.folders = filterFolders(m.allFolders, m.search.Value())

	m.viewport.SetContent(renderFolders(m.folders))

	return m, cmd
}



func (m model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("CMDO ▸ Command History\n"))
	b.WriteString("Search: " + m.search.View() + "\n")
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	b.WriteString(m.viewport.View())

	b.WriteString("\n" + strings.Repeat("─", m.width))

	if time.Since(m.statusAt) < 3*time.Second {
		b.WriteString("\n" + ok.Render(m.status))
	}

	return b.String()
}



func renderFolders(folders []Folder) string {
	var b strings.Builder

	if len(folders) == 0 {
		return muted.Render("No matching commands.\n")
	}

	for _, folder := range folders {
		b.WriteString(fmt.Sprintf("▼ %s (%d)\n", folder.Name, len(folder.Commands)))

		for _, c := range folder.Commands {
			icon := ok.Render("✓")
			if c.ExitCode != 0 {
				icon = bad.Render("✗")
			}

			b.WriteString(fmt.Sprintf(
				"   %-40s %s %s\n",
				c.Text,
				icon,
				c.Timestamp.Format("15:04 02 Jan"),
			))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func loadFolders(db *sql.DB) []Folder {
	rows, err := db.Query(`
		SELECT command, exit_code, timestamp, directory
		FROM commands
		ORDER BY timestamp DESC
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	m := map[string][]Command{}

	for rows.Next() {
		var c Command
		var ts string
		rows.Scan(&c.Text, &c.ExitCode, &ts, &c.Folder)
		c.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		m[c.Folder] = append(m[c.Folder], c)
	}

	var folders []Folder
	for k, v := range m {
		folders = append(folders, Folder{
			Name:     k,
			Commands: v,
			Expanded: true,
		})
	}

	return folders
}

func filterFolders(all []Folder, q string) []Folder {
	if q == "" {
		return all
	}
	q = strings.ToLower(q)

	var res []Folder
	for _, f := range all {
		if strings.Contains(strings.ToLower(f.Name), q) {
			res = append(res, f)
			continue
		}

		var cmds []Command
		for _, c := range f.Commands {
			if strings.Contains(strings.ToLower(c.Text), q) {
				cmds = append(cmds, c)
			}
		}

		if len(cmds) > 0 {
			res = append(res, Folder{
				Name:     f.Name,
				Commands: cmds,
				Expanded: true,
			})
		}
	}
	return res
}
