package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWizardChecksThePortWhileTyping(t *testing.T) {
	busy := func(port int) bool { return port != 8069 }
	var current tea.Model = newModelWith(Options{DockerMissing: true, PortIsFree: busy})
	current = press(current, tea.KeyEnter, tea.KeyEnter, tea.KeyEnter, tea.KeyEnter) // dev, version, folder, name
	current = press(current, tea.KeyEnter, tea.KeyEnter)                             // no Enterprise, no custom repository

	m := current.(model)
	if m.current().key != "port" || m.input != "8069" {
		t.Fatalf("question = %q, input = %q; want the port question pre-filled with 8069", m.current().key, m.input)
	}
	if m.noteOK || !strings.Contains(m.View(), "already in use") {
		t.Fatalf("a busy port must show a warning: note = %q", m.note)
	}

	current = press(current, tea.KeyEnter)
	if current.(model).current().key != "port" {
		t.Fatal("Enter must not accept a busy port")
	}

	current = press(current, tea.KeyBackspace, tea.KeyBackspace, tea.KeyBackspace)
	current = typeText(current, "169")
	m = current.(model)
	if !m.noteOK || !strings.Contains(m.View(), "Port 8169 is free") {
		t.Fatalf("a free port must be confirmed: note = %q", m.note)
	}

	current = press(current, tea.KeyEnter)
	m = current.(model)
	if m.answers["port"] != "8169" || m.current().key != "docker" {
		t.Fatalf("port = %q, question = %q; want 8169 and the Docker question", m.answers["port"], m.current().key)
	}

	current = press(current, tea.KeyDown, tea.KeyEnter) // do not install Docker
	if got := current.(model).current().key; got != "confirm" {
		t.Fatalf("question = %q, want confirm: starting Odoo makes no sense without Docker", got)
	}
}

func TestWizardAsksToStartWhenDockerIsPresent(t *testing.T) {
	var current tea.Model = newModelWith(Options{})
	current = press(current, tea.KeyEnter, tea.KeyEnter, tea.KeyEnter, tea.KeyEnter)
	current = press(current, tea.KeyEnter, tea.KeyEnter)
	current = press(current, tea.KeyEnter) // port 8069

	m := current.(model)
	if m.current().key != "start" {
		t.Fatalf("question = %q, want start; the Docker question must be skipped", m.current().key)
	}
	current = press(current, tea.KeyEnter)
	if got := current.(model).answers["start"]; got != optStartNow {
		t.Fatalf("start = %q, want %q", got, optStartNow)
	}
}
