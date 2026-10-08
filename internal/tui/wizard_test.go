package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func press(current tea.Model, keys ...tea.KeyType) tea.Model {
	for _, key := range keys {
		current, _ = current.Update(tea.KeyMsg{Type: key})
	}
	return current
}

func typeText(current tea.Model, text string) tea.Model {
	current, _ = current.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	return current
}

func TestWizardCommunityPath(t *testing.T) {
	var current tea.Model = newModel()
	current = press(current, tea.KeyDown, tea.KeyEnter) // staging
	current = press(current, tea.KeyEnter)              // 19.0
	current = press(current, tea.KeyEnter)              // path left empty
	current = typeText(current, "shop")
	current = press(current, tea.KeyEnter) // name
	current = press(current, tea.KeyEnter) // no Enterprise
	current = press(current, tea.KeyEnter) // no custom repository

	m := current.(model)
	if m.current().key != "confirm" {
		t.Fatalf("on question %q, want confirm: the token and branch questions must be skipped", m.current().key)
	}
	current = press(current, tea.KeyEnter) // start
	m = current.(model)
	if !m.finished || m.answers["environment"] != "staging" || m.answers["name"] != "shop" {
		t.Fatalf("finished = %v, answers = %v", m.finished, m.answers)
	}
}

func TestWizardHidesTheToken(t *testing.T) {
	var current tea.Model = newModel()
	current = press(current, tea.KeyEnter, tea.KeyEnter, tea.KeyEnter, tea.KeyEnter) // dev, 19.0, path, name
	current = press(current, tea.KeyDown, tea.KeyEnter)                              // Enterprise by token
	current = typeText(current, "secret123")

	m := current.(model)
	if m.current().key != "enterpriseToken" {
		t.Fatalf("on question %q, want enterpriseToken", m.current().key)
	}
	if strings.Contains(m.View(), "secret123") {
		t.Fatal("the token is visible on screen")
	}
	current = press(current, tea.KeyBackspace, tea.KeyEnter)
	if got := current.(model).answers["enterpriseToken"]; got != "secret12" {
		t.Fatalf("token = %q, want secret12", got)
	}
}

func TestWizardMouseAndBack(t *testing.T) {
	var current tea.Model = newModel()
	hover := tea.MouseMsg{X: 5, Y: firstOptionRow + 2, Action: tea.MouseActionMotion}
	current, _ = current.Update(hover)
	if current.(model).cursor != 2 {
		t.Fatalf("cursor = %d, want 2 after hovering the third option", current.(model).cursor)
	}

	click := tea.MouseMsg{X: 5, Y: firstOptionRow + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	current, _ = current.Update(click)
	m := current.(model)
	if m.answers["environment"] != "production" || m.current().key != "version" {
		t.Fatalf("after the click: answers = %v, question = %q", m.answers, m.current().key)
	}

	current = press(current, tea.KeyEsc)
	m = current.(model)
	if m.current().key != "environment" || m.cursor != 2 {
		t.Fatalf("after Esc: question = %q, cursor = %d; want environment with production selected", m.current().key, m.cursor)
	}
}

func TestOptionRowsMatchTheScreen(t *testing.T) {
	m := newModel()
	lines := strings.Split(m.View(), "\n")
	if !strings.Contains(lines[m.optionStart()], "development") {
		t.Fatalf("row %d = %q, want the first option", m.optionStart(), lines[m.optionStart()])
	}

	m.index = len(m.questions) - 1
	m.load()
	lines = strings.Split(m.View(), "\n")
	if !strings.Contains(lines[m.optionStart()], optStart) {
		t.Fatalf("row %d = %q, want the first option of the review screen", m.optionStart(), lines[m.optionStart()])
	}
}
