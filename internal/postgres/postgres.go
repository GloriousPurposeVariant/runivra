// Package postgres finds, installs and prepares a PostgreSQL server that runs
// directly on this computer, outside Docker.
package postgres

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	Version = "17"
	Port    = 5432
	Role    = "odoo"
)

// Status is what was found on this computer.
type Status struct {
	Psql    string // full path of the psql program; "" when PostgreSQL is not installed
	Running bool   // something is accepting connections on the PostgreSQL port
}

func (s Status) Installed() bool {
	return s.Psql != ""
}

// psqlPatterns lists where installers put psql when it is not on the PATH.
func psqlPatterns(system string) []string {
	switch system {
	case "windows":
		return []string{`C:\Program Files\PostgreSQL\*\bin\psql.exe`}
	case "darwin":
		return []string{"/opt/homebrew/opt/postgresql@*/bin/psql", "/usr/local/opt/postgresql@*/bin/psql"}
	default:
		return []string{"/usr/lib/postgresql/*/bin/psql"}
	}
}

func findPsql(system string) string {
	if found, err := exec.LookPath("psql"); err == nil {
		return found
	}
	for _, pattern := range psqlPatterns(system) {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			sort.Strings(matches)
			return matches[len(matches)-1]
		}
	}
	return ""
}

func portOpen(port int) bool {
	connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		return false
	}
	connection.Close()
	return true
}

func Detect() Status {
	return Status{Psql: findPsql(runtime.GOOS), Running: portOpen(Port)}
}

// NewPassword returns a random password made only of letters and digits, so
// it is safe to pass on a command line and inside a SQL statement.
func NewPassword() string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		panic(err)
	}
	return hex.EncodeToString(raw)
}

// installCommands returns the commands that install and start PostgreSQL.
// has reports whether a program is available, so tests can pretend.
func installCommands(system string, root bool, has func(name string) bool, adminPassword string) ([][]string, error) {
	sudo := func(args ...string) []string {
		if root {
			return args
		}
		return append([]string{"sudo"}, args...)
	}
	switch {
	case system == "windows" && has("winget"):
		return [][]string{{
			"winget", "install", "--exact", "--id", "PostgreSQL.PostgreSQL." + Version,
			"--accept-source-agreements", "--accept-package-agreements",
			"--override", "--mode unattended --unattendedmodeui minimal --serverport " + fmt.Sprint(Port) + " --superpassword " + adminPassword,
		}}, nil
	case system == "darwin" && has("brew"):
		return [][]string{
			{"brew", "install", "postgresql@" + Version},
			{"brew", "services", "start", "postgresql@" + Version},
		}, nil
	case system == "linux" && has("apt-get"):
		return [][]string{
			sudo("apt-get", "update"),
			sudo("apt-get", "install", "-y", "postgresql"),
		}, nil
	case system == "linux" && has("dnf"):
		return [][]string{
			sudo("dnf", "install", "-y", "postgresql-server", "postgresql-contrib"),
			sudo("postgresql-setup", "--initdb"),
			sudo("systemctl", "enable", "--now", "postgresql"),
		}, nil
	}
	return nil, fmt.Errorf("automatic install is not supported on this system; install PostgreSQL from https://www.postgresql.org/download/ and run this again")
}

// Install runs the installer, passing its questions and progress straight
// through, then waits for the server to accept connections.
func Install(ctx context.Context, in io.Reader, out io.Writer, adminPassword string) error {
	has := func(name string) bool {
		_, err := exec.LookPath(name)
		return err == nil
	}
	commands, err := installCommands(runtime.GOOS, os.Geteuid() == 0, has, adminPassword)
	if err != nil {
		return err
	}
	for _, command := range commands {
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		cmd.Stdin = in
		cmd.Stdout = out
		cmd.Stderr = out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s failed: %w", command[0], err)
		}
	}
	for attempt := 0; attempt < 60; attempt++ {
		if portOpen(Port) {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("cancelled")
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("PostgreSQL was installed but is not accepting connections on port %d yet", Port)
}

// adminCommand builds the command that runs SQL as PostgreSQL's administrator.
// How that administrator is reached differs per system.
func adminCommand(system string, root bool, psql string, adminPassword string, sql string) ([]string, []string) {
	switch system {
	case "windows":
		return []string{psql, "-U", "postgres", "-h", "127.0.0.1", "-p", fmt.Sprint(Port), "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", sql},
			[]string{"PGPASSWORD=" + adminPassword}
	case "darwin":
		return []string{psql, "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", sql}, nil
	}
	if root {
		return []string{"runuser", "-u", "postgres", "--", psql, "-v", "ON_ERROR_STOP=1", "-c", sql}, nil
	}
	return []string{"sudo", "-u", "postgres", psql, "-v", "ON_ERROR_STOP=1", "-c", sql}, nil
}

func run(ctx context.Context, args []string, env []string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(os.Environ(), env...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message == "" {
			message = err.Error()
		}
		return errors.New(message)
	}
	return nil
}

// CreateRole makes the database user Odoo connects as. Odoo creates its own
// databases, so the user is allowed to do that and nothing more.
func CreateRole(ctx context.Context, psql string, adminPassword string, role string, password string) error {
	sql := fmt.Sprintf("CREATE ROLE %s LOGIN CREATEDB PASSWORD '%s'", role, password)
	args, env := adminCommand(runtime.GOOS, os.Geteuid() == 0, psql, adminPassword, sql)
	return run(ctx, args, env)
}

// Check confirms that a user and password can really log in.
func Check(ctx context.Context, psql string, user string, password string) error {
	args := []string{psql, "-U", user, "-h", "127.0.0.1", "-p", fmt.Sprint(Port), "-d", "postgres", "-w", "-At", "-c", "SELECT 1"}
	return run(ctx, args, []string{"PGPASSWORD=" + password})
}
