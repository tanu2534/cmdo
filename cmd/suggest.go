package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type suggestModel struct {
	all      []string
	filtered []string
	input    textinput.Model
	cursor   int
	chosen   string
}

var (
	suggHighlight = lipgloss.NewStyle().Background(lipgloss.Color("63")).Foreground(lipgloss.Color("15"))
	suggMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	suggHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
)

func newSuggestModel(cmds []string, initial string) suggestModel {
	ti := textinput.New()
	ti.Placeholder = "type to search..."
	ti.SetValue(initial)
	ti.Focus()
	ti.Width = 50

	return suggestModel{
		all:      cmds,
		filtered: filterSuggestions(cmds, initial),
		input:    ti,
	}
}

func (m suggestModel) Init() tea.Cmd { return nil }

func (m suggestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 {
				m.chosen = m.filtered[m.cursor]
			}
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		}
	}

	m.input, cmd = m.input.Update(msg)
	m.filtered = filterSuggestions(m.all, m.input.Value())
	if m.cursor >= len(m.filtered) {
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		} else {
			m.cursor = 0
		}
	}
	return m, cmd
}

func (m suggestModel) View() string {
	var b strings.Builder
	dir, _ := os.Getwd()
	b.WriteString(suggHeader.Render("CMDO ▸ Suggest") + "  " + suggMuted.Render(dir) + "\n")
	b.WriteString("Search: " + m.input.View() + "\n")
	b.WriteString(strings.Repeat("─", 60) + "\n")

	if len(m.filtered) == 0 {
		b.WriteString(suggMuted.Render("  no matching commands\n"))
	} else {
		limit := 10
		start := 0
		if m.cursor >= limit {
			start = m.cursor - limit + 1
		}
		end := start + limit
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		for i := start; i < end; i++ {
			line := fmt.Sprintf("  %s", m.filtered[i])
			if i == m.cursor {
				b.WriteString(suggHighlight.Render(line) + "\n")
			} else {
				b.WriteString(line + "\n")
			}
		}
	}

	b.WriteString(strings.Repeat("─", 60) + "\n")
	b.WriteString(suggMuted.Render("↑↓ navigate  enter select  esc quit"))
	return b.String()
}

func filterSuggestions(all []string, q string) []string {
	if q == "" {
		return all
	}
	q = strings.ToLower(q)
	var out []string
	for _, c := range all {
		if strings.Contains(strings.ToLower(c), q) {
			out = append(out, c)
		}
	}
	return out
}

var suggestCmd = &cobra.Command{
	Use:   "suggest [query]",
	Short: "Interactively suggest commands from current directory history",
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := os.Getwd()
		cmds := queryCompletions(strings.Join(args, " "), dir)

		if len(cmds) == 0 {
			fmt.Println("No commands found for", dir)
			return
		}

		initial := ""
		if len(args) > 0 {
			initial = strings.Join(args, " ")
		}

		m := newSuggestModel(cmds, initial)
		p := tea.NewProgram(m, tea.WithAltScreen())
		result, err := p.Run()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if sm, ok := result.(suggestModel); ok && sm.chosen != "" {
			fmt.Println(sm.chosen)
		}
	},
}

func init() {
	rootCmd.AddCommand(suggestCmd)
}
