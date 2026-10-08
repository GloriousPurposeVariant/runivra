package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLongListsScrollToKeepTheSelectionVisible(t *testing.T) {
	root := t.TempDir()
	for number := 1; number <= 30; number++ {
		if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("folder-%02d", number)), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	m := newModel()
	m.index = 2
	m.open(root)
	m.browser.limit = 30

	var current tea.Model = m
	current, _ = current.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = current.(model)
	room := m.capacity()
	if room >= len(m.options()) {
		t.Fatalf("capacity = %d with %d options; the list should not fit a 20-line window", room, len(m.options()))
	}
	if lines := strings.Count(m.View(), "\n"); lines > 20 {
		t.Fatalf("the screen is %d lines tall, more than the 20-line window", lines)
	}

	for step := 0; step < 25; step++ {
		current = press(current, tea.KeyDown)
	}
	m = current.(model)
	if m.cursor != 25 || m.offset == 0 {
		t.Fatalf("cursor = %d, offset = %d; the list should have scrolled", m.cursor, m.offset)
	}
	if !strings.Contains(m.View(), "❯ "+m.options()[25]) {
		t.Fatal("the selected line is not on screen after scrolling down")
	}

	hover := tea.MouseMsg{X: 5, Y: m.optionStart(), Action: tea.MouseActionMotion}
	current, _ = current.Update(hover)
	m = current.(model)
	if m.cursor != m.offset {
		t.Fatalf("hovering the first visible row selected %d, want %d", m.cursor, m.offset)
	}

	for step := 0; step < 30; step++ {
		current = press(current, tea.KeyUp)
	}
	m = current.(model)
	if m.cursor != 0 || m.offset != 0 || !strings.Contains(m.View(), "❯ "+optUseFolder) {
		t.Fatalf("cursor = %d, offset = %d; the top of the list should be back on screen", m.cursor, m.offset)
	}
}

func TestVersionsComeFromOptions(t *testing.T) {
	m := newModelWith(Options{Versions: []string{"20.0", "19.0"}})
	m.index = 1
	if got := m.options(); len(got) != 2 || got[0] != "20.0" {
		t.Fatalf("version options = %q, want the two given versions", got)
	}
}
