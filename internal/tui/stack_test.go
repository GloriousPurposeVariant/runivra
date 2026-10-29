package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// toDatabasePlace answers everything up to "Where is the database?", with
// Odoo in Docker.
func toDatabasePlace(options Options) tea.Model {
	var current tea.Model = newModelWith(options)
	current = press(current, tea.KeyEnter, tea.KeyEnter) // development, version
	current = press(current, tea.KeyEnter)               // Odoo in Docker
	current = press(current, tea.KeyEnter, tea.KeyEnter) // folder, name
	current = press(current, tea.KeyEnter, tea.KeyEnter) // no Enterprise, no custom repository
	return current
}

// toDatabaseQuestions answers everything up to the first database question,
// choosing Odoo in Docker and PostgreSQL on this computer.
func toDatabaseQuestions(options Options) tea.Model {
	current := toDatabasePlace(options)
	return press(current, tea.KeyDown, tea.KeyEnter) // on this computer
}

func TestWizardAsksForAnExistingLogin(t *testing.T) {
	current := toDatabaseQuestions(Options{PostgresInstalled: true})
	current = press(current, tea.KeyEnter) // use an existing user
	m := current.(model)

	if m.current().key != "dbUser" || m.input != "odoo" {
		t.Fatalf("question = %q, input = %q; want the user question pre-filled with odoo", m.current().key, m.input)
	}

	current = press(current, tea.KeyBackspace, tea.KeyBackspace, tea.KeyBackspace, tea.KeyBackspace)
	current = typeText(current, "postgres")
	current = press(current, tea.KeyEnter)
	if got := current.(model).current().key; got != "dbUser" {
		t.Fatalf("question = %q; postgres must be refused as the Odoo user", got)
	}

	for range "postgres" {
		current = press(current, tea.KeyBackspace)
	}
	current = typeText(current, "shop")
	current = press(current, tea.KeyEnter)
	m = current.(model)
	if m.current().key != "dbPassword" {
		t.Fatalf("question = %q, want dbPassword", m.current().key)
	}

	current = press(current, tea.KeyEnter)
	if got := current.(model).current().key; got != "dbPassword" {
		t.Fatalf("question = %q; an existing user needs a password", got)
	}

	current = typeText(current, "s3cret")
	if strings.Contains(current.(model).View(), "s3cret") {
		t.Fatal("the database password is visible on screen")
	}
	current = press(current, tea.KeyEnter)
	m = current.(model)
	if m.answers["dbUser"] != "shop" || m.answers["dbPassword"] != "s3cret" || m.current().key != "port" {
		t.Fatalf("answers = %v, question = %q", m.answers, m.current().key)
	}
}

func TestWizardOffersToInstallPostgres(t *testing.T) {
	current := toDatabaseQuestions(Options{PostgresInstalled: false})
	if got := current.(model).current().key; got != "postgres" {
		t.Fatalf("question = %q, want the offer to install PostgreSQL", got)
	}

	current = press(current, tea.KeyEnter) // yes, install
	current = press(current, tea.KeyEnter) // user odoo
	m := current.(model)
	if m.current().key != "dbPassword" {
		t.Fatalf("question = %q, want dbPassword", m.current().key)
	}
	current = press(current, tea.KeyEnter) // empty: generate one
	m = current.(model)
	if m.current().key != "port" || m.answers["dbPassword"] != "" {
		t.Fatalf("question = %q, password = %q; an empty password must be accepted for a new install", m.current().key, m.answers["dbPassword"])
	}
}

func TestWizardSkipsTheLoginWhenInstallIsDeclined(t *testing.T) {
	current := toDatabaseQuestions(Options{PostgresInstalled: false})
	current = press(current, tea.KeyDown, tea.KeyEnter) // no
	if got := current.(model).current().key; got != "port" {
		t.Fatalf("question = %q, want port: there is nothing to log in to", got)
	}
}

func TestDockerStackAsksNoDatabaseQuestions(t *testing.T) {
	current := toDatabasePlace(Options{PostgresInstalled: true})
	current = press(current, tea.KeyEnter) // in Docker, next to Odoo
	if got := current.(model).current().key; got != "port" {
		t.Fatalf("question = %q, want port", got)
	}
}
