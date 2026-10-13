// Package database works with the PostgreSQL databases of a project.
package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GloriousPurposeVariant/runivra/internal/odooconf"
	"github.com/GloriousPurposeVariant/runivra/internal/postgres"
	"github.com/GloriousPurposeVariant/runivra/internal/workspace"
)

// ErrNotRunning means PostgreSQL is not up, so nothing can be asked of it.
var ErrNotRunning = errors.New("the database is not running")

// The same rule Odoo's own database screen uses: databases owned by the user
// Odoo connects as, leaving out PostgreSQL's built-in ones.
const listQuery = `SELECT datname FROM pg_database
WHERE NOT datistemplate AND datname <> 'postgres'
AND datdba = (SELECT usesysid FROM pg_user WHERE usename = current_user)
ORDER BY datname`

// Connection is how to reach a project's PostgreSQL from this computer.
type Connection struct {
	Service  string // the compose service; "" when PostgreSQL runs on this computer
	Compose  string // the compose file, used only with a service
	Host     string
	Port     string
	User     string
	Password string
}

// ConnectionFor reads the database settings from the project's odoo.conf.
func ConnectionFor(view workspace.View) Connection {
	values := map[string]string{}
	if view.ConfPath != "" {
		if read, err := odooconf.Read(view.ConfPath); err == nil {
			values = read
		}
	}
	connection := Connection{
		Service:  view.Config.Docker.DatabaseService,
		Compose:  filepath.Join(view.Root, filepath.FromSlash(view.Config.Docker.ComposeFile)),
		Host:     localHost(values["db_host"]),
		Port:     values["db_port"],
		User:     values["db_user"],
		Password: values["db_password"],
	}
	if connection.Port == "" || connection.Port == "False" {
		connection.Port = "5432"
	}
	if connection.User == "" {
		connection.User = "odoo"
	}
	return connection
}

// localHost turns the name Odoo uses from inside its container into the name
// that works from this computer. odoo.conf is written from Odoo's side.
func localHost(host string) string {
	switch host {
	case "", "False", "localhost", "host.docker.internal":
		return "127.0.0.1"
	}
	return host
}

// command builds the psql call for a query, and the environment it needs.
func (c Connection) command(psql string, query string) ([]string, []string) {
	if c.Service != "" {
		return []string{"docker", "compose", "-f", c.Compose, "exec", "-T", c.Service,
			"psql", "-U", c.User, "-d", "postgres", "-At", "-c", query}, nil
	}
	return []string{psql, "-U", c.User, "-h", c.Host, "-p", c.Port, "-d", "postgres", "-w", "-At", "-c", query},
		[]string{"PGPASSWORD=" + c.Password}
}

func parseNames(output string) []string {
	var names []string
	for _, line := range strings.Split(output, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// List returns the project's databases, asking PostgreSQL inside its
// container or on this computer, whichever the project uses.
func List(ctx context.Context, view workspace.View) ([]string, error) {
	connection := ConnectionFor(view)

	psql := ""
	if connection.Service == "" {
		psql = postgres.Detect().Psql
		if psql == "" {
			return nil, errors.New("this project uses PostgreSQL on this computer, but psql was not found; run \"runivra postgres\"")
		}
	}

	args, env := connection.command(psql, listQuery)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(os.Environ(), env...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if strings.Contains(message, "is not running") || strings.Contains(message, "Connection refused") {
			return nil, ErrNotRunning
		}
		return nil, fmt.Errorf("could not list databases: %s", message)
	}
	return parseNames(string(output)), nil
}

// Choose decides which database a command should work on. An explicit name
// wins, then the remembered one, then the only one there is. It returns ""
// when the caller has to ask.
func Choose(names []string, requested string, remembered string) (string, error) {
	contains := func(name string) bool {
		for _, candidate := range names {
			if candidate == name {
				return true
			}
		}
		return false
	}
	if requested != "" {
		if !contains(requested) {
			return "", fmt.Errorf("there is no database named %q; available: %s", requested, strings.Join(names, ", "))
		}
		return requested, nil
	}
	if remembered != "" && contains(remembered) {
		return remembered, nil
	}
	if len(names) == 1 {
		return names[0], nil
	}
	return "", nil
}
