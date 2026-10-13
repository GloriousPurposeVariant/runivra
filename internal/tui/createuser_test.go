package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWizardChecksAnExistingLoginOnTheSpot(t *testing.T) {
	options := Options{
		PostgresInstalled: true,
		CheckLogin: func(user string, password string) error {
			if user == "odoo" && password == "right" {
				return nil
			}
			return errors.New("That password is not right for user " + user + ".")
		},
	}
	current := toDatabaseQuestions(options)
	current = press(current, tea.KeyEnter) // use an existing user
	current = press(current, tea.KeyEnter) // odoo
	current = typeText(current, "wrong")
	current = press(current, tea.KeyEnter)

	m := current.(model)
	if m.current().key != "dbPassword" || !strings.Contains(m.View(), "not right for user odoo") {
		t.Fatalf("question = %q, note = %q; a wrong password must be reported on the password screen", m.current().key, m.note)
	}

	for range "wrong" {
		current = press(current, tea.KeyBackspace)
	}
	current = typeText(current, "right")
	current = press(current, tea.KeyEnter)
	m = current.(model)
	if m.current().key != "port" || m.answers["dbPassword"] != "right" {
		t.Fatalf("question = %q; the right password must be accepted", m.current().key)
	}
}

func TestWizardCreatesAUserInAnExistingPostgres(t *testing.T) {
	options := Options{
		PostgresInstalled:  true,
		NeedsAdminPassword: true,
		CheckAdmin: func(password string) error {
			if password == "admin-pass" {
				return nil
			}
			return errors.New("That password is not right for the postgres administrator.")
		},
		RoleExists: func(adminPassword string, name string) (bool, error) {
			return name == "odoo", nil
		},
	}
	current := toDatabaseQuestions(options)
	current = press(current, tea.KeyDown, tea.KeyEnter) // create a new user

	m := current.(model)
	if m.current().key != "dbAdmin" {
		t.Fatalf("question = %q, want the administrator password first", m.current().key)
	}
	current = typeText(current, "nope")
	current = press(current, tea.KeyEnter)
	if got := current.(model).current().key; got != "dbAdmin" {
		t.Fatalf("question = %q; a wrong administrator password must be refused", got)
	}
	for range "nope" {
		current = press(current, tea.KeyBackspace)
	}
	current = typeText(current, "admin-pass")
	current = press(current, tea.KeyEnter)

	m = current.(model)
	if m.current().key != "dbUser" || m.current().title != "Name for the new database user" {
		t.Fatalf("question = %q (%q), want the new user's name", m.current().key, m.current().title)
	}
	current = press(current, tea.KeyEnter) // odoo, which is taken
	m = current.(model)
	if m.current().key != "dbUser" || !strings.Contains(m.note, "already exists") {
		t.Fatalf("question = %q, note = %q; a name that is taken must be refused on this screen", m.current().key, m.note)
	}

	current = typeText(current, "_dev")
	current = press(current, tea.KeyEnter) // odoo_dev is free
	current = press(current, tea.KeyEnter) // empty password: generate one
	m = current.(model)
	if m.current().key != "port" || m.answers["dbUser"] != "odoo_dev" || m.answers["dbAdmin"] != "admin-pass" {
		t.Fatalf("question = %q, answers = %v", m.current().key, m.answers)
	}
}

func TestWizardNeedsNoAdminPasswordWhereTheSystemProvidesOne(t *testing.T) {
	current := toDatabaseQuestions(Options{PostgresInstalled: true, NeedsAdminPassword: false})
	current = press(current, tea.KeyDown, tea.KeyEnter) // create a new user
	current = press(current, tea.KeyEnter)              // odoo
	current = press(current, tea.KeyEnter)              // generate the password
	if got := current.(model).current().key; got != "port" {
		t.Fatalf("question = %q, want port: Linux and macOS reach the administrator without a password", got)
	}
}
