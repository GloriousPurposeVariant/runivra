package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// toRemoteServer chooses a database on another server and describes it:
// db.example.com, the default port, and SSL required.
func toRemoteServer(options Options) tea.Model {
	current := toDatabasePlace(options)
	current = press(current, tea.KeyDown, tea.KeyDown, tea.KeyEnter) // another server
	current = typeText(current, "db.example.com")
	current = press(current, tea.KeyEnter) // host
	current = press(current, tea.KeyEnter) // port 5432
	current = press(current, tea.KeyEnter) // require SSL
	return current
}

func TestRemoteDatabaseAsksForTheServer(t *testing.T) {
	current := toDatabasePlace(Options{})
	current = press(current, tea.KeyDown, tea.KeyDown, tea.KeyEnter) // another server

	m := current.(model)
	if m.current().key != "dbHost" {
		t.Fatalf("question = %q, want the server's address", m.current().key)
	}
	current = press(current, tea.KeyEnter)
	if got := current.(model).current().key; got != "dbHost" {
		t.Fatalf("question = %q; an empty address must be refused", got)
	}

	current = typeText(current, "db.example.com")
	current = press(current, tea.KeyEnter)
	m = current.(model)
	if m.current().key != "dbPort" || m.input != "5432" {
		t.Fatalf("question = %q, input = %q; want the port, pre-filled with 5432", m.current().key, m.input)
	}
	current = press(current, tea.KeyEnter)
	if got := current.(model).current().key; got != "dbSSL" {
		t.Fatalf("question = %q, want the SSL question", got)
	}
	current = press(current, tea.KeyEnter) // require SSL
	m = current.(model)
	if m.current().key != "dbMode" || m.answers["postgres"] != "" {
		t.Fatalf("question = %q; nothing may be installed for a remote server", m.current().key)
	}
}

func TestRemoteDatabaseChecksAnExistingLogin(t *testing.T) {
	var seen Remote
	options := Options{
		PostgresInstalled: false, // must not lead to an offer to install it
		CheckRemoteLogin: func(remote Remote, user string, password string) error {
			seen = remote
			if user == "odoo" && password == "right" {
				return nil
			}
			return errors.New("password authentication failed for user \"" + user + "\"")
		},
	}
	current := toRemoteServer(options)
	current = press(current, tea.KeyEnter) // use an existing user
	current = press(current, tea.KeyEnter) // odoo
	current = typeText(current, "wrong")
	current = press(current, tea.KeyEnter)

	m := current.(model)
	if m.current().key != "dbPassword" || !strings.Contains(m.View(), "password authentication failed") {
		t.Fatalf("question = %q, note = %q; a wrong password must be reported on the spot", m.current().key, m.note)
	}
	if seen != (Remote{Host: "db.example.com", Port: 5432, SSLMode: "require"}) {
		t.Fatalf("checked against %+v, want the server from the earlier answers", seen)
	}

	for range "wrong" {
		current = press(current, tea.KeyBackspace)
	}
	current = typeText(current, "right")
	current = press(current, tea.KeyEnter)
	m = current.(model)
	if m.current().key != "port" {
		t.Fatalf("question = %q, want port", m.current().key)
	}
}

func TestRemoteDatabaseCreatesAUserThroughTheMainUser(t *testing.T) {
	options := Options{
		CheckRemoteAdmin: func(remote Remote, user string, password string) error {
			if user == "postgres" && password == "main-pass" {
				return nil
			}
			return errors.New("password authentication failed for user \"" + user + "\"")
		},
		RemoteRoleExists: func(remote Remote, adminUser string, adminPassword string, name string) (bool, error) {
			return name == "odoo", nil
		},
	}
	current := toRemoteServer(options)
	current = press(current, tea.KeyDown, tea.KeyEnter) // create a new user

	m := current.(model)
	if m.current().key != "dbAdminUser" || m.input != "postgres" {
		t.Fatalf("question = %q, input = %q; want the main user, pre-filled with postgres", m.current().key, m.input)
	}
	current = press(current, tea.KeyEnter)
	current = typeText(current, "main-pass")
	current = press(current, tea.KeyEnter)

	m = current.(model)
	if m.current().key != "dbUser" || m.current().title != "Name for the new database user" {
		t.Fatalf("question = %q (%q), want the new user's name", m.current().key, m.current().title)
	}
	current = press(current, tea.KeyEnter) // odoo, which is taken
	if note := current.(model).note; !strings.Contains(note, "already exists") {
		t.Fatalf("note = %q; a name that is taken must be refused", note)
	}
	current = typeText(current, "_shop")
	current = press(current, tea.KeyEnter) // odoo_shop
	current = press(current, tea.KeyEnter) // generate the password
	current = press(current, tea.KeyEnter) // port 8069
	current = press(current, tea.KeyEnter) // start

	m = current.(model)
	if m.current().key != "confirm" || !strings.Contains(m.View(), "db.example.com:5432, user odoo_shop (will be created), SSL require") {
		t.Fatalf("question = %q; the review must describe the server:\n%s", m.current().key, m.View())
	}
	current = press(current, tea.KeyEnter) // start setup
	m = current.(model)
	if !m.finished || databasePlace(m.answers) != "remote" || m.answers["dbAdminUser"] != "postgres" {
		t.Fatalf("finished = %v, answers = %v", m.finished, m.answers)
	}
}

func TestHostOdooCannotPutTheDatabaseInDocker(t *testing.T) {
	var current tea.Model = newModelWith(Options{PostgresInstalled: true})
	current = press(current, tea.KeyEnter, tea.KeyEnter) // development, version
	current = press(current, tea.KeyDown, tea.KeyEnter)  // Odoo on this computer
	current = press(current, tea.KeyEnter, tea.KeyEnter) // folder, name
	current = press(current, tea.KeyEnter, tea.KeyEnter) // no Enterprise, no custom repository

	m := current.(model)
	if m.current().key != "database" || len(m.options()) != 2 || m.options()[0] != optDBLocal {
		t.Fatalf("question = %q, options = %v; with Odoo on this computer there is no compose file for PostgreSQL", m.current().key, m.options())
	}
	current = press(current, tea.KeyDown, tea.KeyEnter) // another server
	if got := current.(model).current().key; got != "dbHost" {
		t.Fatalf("question = %q, want the server's address", got)
	}
}

func TestSSLMode(t *testing.T) {
	for answer, want := range map[string]string{optSSLRequire: "require", optSSLPrefer: "", optSSLDisable: "disable"} {
		if got := sslMode(answer); got != want {
			t.Errorf("sslMode(%q) = %q, want %q", answer, got, want)
		}
	}
}
