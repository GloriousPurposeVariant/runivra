// Package database works with the PostgreSQL databases of a project.
package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
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
	Service string // the compose service; "" when PostgreSQL is reached over the network
	Compose string // the compose file, used only with a service
	Server  postgres.Server
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
		Service: view.Config.Docker.DatabaseService,
		Compose: filepath.Join(view.Root, filepath.FromSlash(view.Config.Docker.ComposeFile)),
		Server: postgres.Server{
			Host:     localHost(values["db_host"]),
			Port:     postgres.Port,
			User:     unset(values["db_user"]),
			Password: unset(values["db_password"]),
			SSLMode:  unset(values["db_sslmode"]),
		},
	}
	if port, err := strconv.Atoi(values["db_port"]); err == nil {
		connection.Server.Port = port
	}
	if connection.Server.User == "" {
		connection.Server.User = "odoo"
	}
	return connection
}

// unset treats Odoo's way of writing "no value" as empty.
func unset(value string) string {
	if value == "False" {
		return ""
	}
	return value
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

// command builds the psql call that runs a query inside the database
// container, whose port is not published on this computer.
func (c Connection) command(query string) []string {
	return []string{"docker", "compose", "-f", c.Compose, "exec", "-T", c.Service,
		"psql", "-U", c.Server.User, "-d", "postgres", "-At", "-c", query}
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
// container, or wherever odoo.conf says it is.
func List(ctx context.Context, view workspace.View) ([]string, error) {
	connection := ConnectionFor(view)
	if connection.Service == "" {
		names, err := connection.Server.Strings(ctx, listQuery)
		if errors.Is(err, postgres.ErrUnreachable) {
			return nil, ErrNotRunning
		}
		if err != nil {
			return nil, fmt.Errorf("could not list databases: %w", err)
		}
		return names, nil
	}

	args := connection.command(listQuery)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if strings.Contains(message, "is not running") {
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
