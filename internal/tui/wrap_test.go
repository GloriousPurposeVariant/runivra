package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWrap(t *testing.T) {
	text := "Typing is hidden. It is checked now and only written to this project's odoo.conf."
	lines := wrap(text, 30)
	if len(lines) < 3 {
		t.Fatalf("lines = %q, want the text split over several lines", lines)
	}
	for _, line := range lines {
		if len([]rune(line)) > 30 {
			t.Fatalf("line %q is wider than 30", line)
		}
	}
	if strings.Join(lines, " ") != text {
		t.Fatalf("wrapping changed the words: %q", lines)
	}

	if got := wrap(text, 0); len(got) != 1 || got[0] != text {
		t.Fatalf("an unknown width must leave the text alone, got %q", got)
	}
	if got := wrap(strings.Repeat("x", 25), 10); len(got) != 3 {
		t.Fatalf("a word longer than the width must be cut, got %q", got)
	}
}

func TestLongHelpWrapsAndKeepsClicksAligned(t *testing.T) {
	m := newModel()
	m.questions[0].help = strings.Repeat("several words of help ", 8)

	var current tea.Model = m
	current, _ = current.Update(tea.WindowSizeMsg{Width: 40, Height: 30})
	m = current.(model)

	lines := strings.Split(m.View(), "\n")
	for _, line := range lines {
		if strings.Contains(line, "words of help") && len([]rune(line)) > 40 {
			t.Fatalf("help line is wider than the 40-column window: %q", line)
		}
	}
	if len(m.helpLines()) < 2 {
		t.Fatalf("help = %q, want it wrapped", m.helpLines())
	}
	if !strings.Contains(lines[m.optionStart()], "development") {
		t.Fatalf("row %d = %q, want the first option; wrapped help must move the click rows with it", m.optionStart(), lines[m.optionStart()])
	}
}
