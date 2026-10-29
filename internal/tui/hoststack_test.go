package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// toHostStack answers up to the first question after the sources, choosing
// "everything on this computer".
func toHostStack(options Options) tea.Model {
	var current tea.Model = newModelWith(options)
	current = press(current, tea.KeyEnter, tea.KeyEnter) // development, version
	current = press(current, tea.KeyDown, tea.KeyEnter)  // Odoo on this computer
	current = press(current, tea.KeyEnter, tea.KeyEnter) // folder, name
	current = press(current, tea.KeyEnter, tea.KeyEnter) // no Enterprise, no custom repository
	current = press(current, tea.KeyEnter)               // database on this computer
	return current

}

func TestHostStackAsksAboutPythonAndNeverAboutDocker(t *testing.T) {
	options := Options{
		DockerMissing:     true,
		PostgresInstalled: true,
		PythonMissing:     func(odooVersion string) bool { return true },
	}
	current := toHostStack(options)
	current = press(current, tea.KeyEnter) // use an existing database user
	current = press(current, tea.KeyEnter) // odoo
	current = typeText(current, "secret")
	current = press(current, tea.KeyEnter)

	m := current.(model)
	if m.current().key != "python" {
		t.Fatalf("question = %q, want the offer to install Python", m.current().key)
	}
	current = press(current, tea.KeyEnter) // yes, install
	current = press(current, tea.KeyEnter) // port

	m = current.(model)
	if m.current().key != "start" {
		t.Fatalf("question = %q, want start: Docker is not needed for this stack, so it must not be asked about", m.current().key)
	}
	current = press(current, tea.KeyEnter)
	m = current.(model)
	if m.current().key != "confirm" || !strings.Contains(m.View(), "on this computer, from a Python environment") {
		t.Fatalf("question = %q; the review must say how Odoo will run", m.current().key)
	}
	if m.answers["python"] != optInstallPy || m.answers["stack"] != optStackHost {
		t.Fatalf("answers = %v", m.answers)
	}
}

func TestHostStackSkipsThePythonQuestionWhenItIsInstalled(t *testing.T) {
	options := Options{
		PostgresInstalled: true,
		PythonMissing:     func(odooVersion string) bool { return false },
	}
	current := toHostStack(options)
	current = press(current, tea.KeyEnter, tea.KeyEnter) // existing user, odoo
	current = typeText(current, "secret")
	current = press(current, tea.KeyEnter)
	if got := current.(model).current().key; got != "port" {
		t.Fatalf("question = %q, want port", got)
	}
}

func TestDockerStacksNeverAskAboutPython(t *testing.T) {
	asked := false
	options := Options{PythonMissing: func(odooVersion string) bool { asked = true; return true }}
	var current tea.Model = newModelWith(options)
	current = press(current, tea.KeyEnter, tea.KeyEnter, tea.KeyEnter) // development, version, Odoo in Docker
	current = press(current, tea.KeyEnter, tea.KeyEnter)               // folder, name
	current = press(current, tea.KeyEnter, tea.KeyEnter)               // no Enterprise, no custom repository
	current = press(current, tea.KeyEnter)                             // database in Docker
	if got := current.(model).current().key; got != "port" || asked {
		t.Fatalf("question = %q, python checked = %v; a Docker stack has no use for Python on this computer", got, asked)
	}
}
