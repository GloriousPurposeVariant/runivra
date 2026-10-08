package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestArrowKeysAndEnterChoose(t *testing.T) {
	var current tea.Model = model{options: []string{"development", "staging", "production"}}
	current, _ = current.Update(tea.KeyMsg{Type: tea.KeyDown})
	current, _ = current.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := current.(model).chosen; got != "staging" {
		t.Fatalf("chosen = %q, want staging", got)
	}
}

func TestClickChooses(t *testing.T) {
	var current tea.Model = model{options: []string{"development", "staging", "production"}}
	click := tea.MouseMsg{X: 3, Y: firstOptionRow + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	current, _ = current.Update(click)
	if got := current.(model).chosen; got != "production" {
		t.Fatalf("chosen = %q, want production", got)
	}
}
