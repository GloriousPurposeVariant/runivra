package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const firstOptionRow = 4

type model struct {
	options []string
	cursor  int
	chosen  string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.options[m.cursor]
			return m, tea.Quit
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			row := msg.Y - firstOptionRow
			if row >= 0 && row < len(m.options) {
				m.cursor = row
				m.chosen = m.options[row]
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString("Runivra setup\n\n")
	b.WriteString("Which environment are you setting up?\n\n")
	for index, option := range m.options {
		marker := "  "
		if index == m.cursor {
			marker = "> "
		}
		b.WriteString(marker + option + "\n")
	}
	b.WriteString("\nUp/Down or click to choose, Enter to confirm, Q to quit\n")
	return b.String()
}

func ChooseEnvironment() (string, error) {
	start := model{options: []string{"development", "staging", "production"}}
	final, err := tea.NewProgram(start, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return "", err
	}
	return final.(model).chosen, nil
}
