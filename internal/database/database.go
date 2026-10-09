// Package database works with the PostgreSQL databases of a project.
package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GloriousPurposeVariant/runivra/internal/odooconf"
	"github.com/GloriousPurposeVariant/runivra/internal/project"
)

// ErrNotRunning means the database container is not up, so nothing can be
// asked of it.
var ErrNotRunning = errors.New("the database is not running")

// The same rule Odoo's own database screen uses: databases owned by the user
// Odoo connects as, leaving out PostgreSQL's built-in ones.
const listQuery = `SELECT datname FROM pg_database
WHERE NOT datistemplate AND datname <> 'postgres'
AND datdba = (SELECT usesysid FROM pg_user WHERE usename = current_user)
ORDER BY datname`

// User returns the PostgreSQL user Odoo connects as, read from odoo.conf.
func User(root string, config project.Config) string {
	values, err := odooconf.Read(filepath.Join(root, filepath.FromSlash(config.Paths.Config)))
	if err != nil || values["db_user"] == "" {
		return "odoo"
	}
	return values["db_user"]
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

// List returns the project's databases by asking PostgreSQL inside the
// database container.
func List(ctx context.Context, root string, config project.Config) ([]string, error) {
	if config.Docker.DatabaseService == "" {
		return nil, errors.New("this project has no database service in its Runivra config; run \"runivra init --force\" to set one")
	}
	composeFile := filepath.Join(root, filepath.FromSlash(config.Docker.ComposeFile))
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile,
		"exec", "-T", config.Docker.DatabaseService,
		"psql", "-U", User(root, config), "-d", "postgres", "-At", "-c", listQuery)
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
