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
	"regexp"
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

// Admin is how Runivra acts as PostgreSQL's administrator, once, to create
// Odoo's database user. On Linux and macOS the local administrator is the
// system's own postgres account, reached through psql without a password.
// Everywhere else it is an ordinary login with a password.
type Admin struct {
	Server Server
	Psql   string // set only for the local administrator on Linux and macOS
}

// LocalAdmin is the administrator of PostgreSQL on this computer. On Windows
// that is the postgres user with the password chosen at install time.
func LocalAdmin(psql string, password string) Admin {
	if runtime.GOOS == "windows" {
		return Admin{Server: Local("postgres", password)}
	}
	return Admin{Psql: psql}
}

// adminCommand builds the psql call that runs SQL as the system's postgres
// account on Linux and macOS.
func adminCommand(system string, root bool, psql string, sql string) []string {
	if system == "darwin" {
		return []string{psql, "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-At", "-c", sql}
	}
	if root {
		return []string{"runuser", "-u", "postgres", "--", psql, "-v", "ON_ERROR_STOP=1", "-At", "-c", sql}
	}
	return []string{"sudo", "-u", "postgres", psql, "-v", "ON_ERROR_STOP=1", "-At", "-c", sql}
}

// query runs a query as the administrator and returns one value per row.
func (a Admin) query(ctx context.Context, sql string) ([]string, error) {
	if a.Psql == "" {
		return a.Server.Strings(ctx, sql)
	}
	output, err := capture(ctx, adminCommand(runtime.GOOS, os.Geteuid() == 0, a.Psql, sql))
	if err != nil {
		return nil, err
	}
	var values []string
	for _, line := range strings.Split(output, "\n") {
		if value := strings.TrimSpace(line); value != "" {
			values = append(values, value)
		}
	}
	return values, nil
}

// capture runs a command and returns what it printed. On failure the error
// carries the program's own message.
func capture(ctx context.Context, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return output.String(), nil
}

// Check confirms that Runivra can act as the administrator.
func (a Admin) Check(ctx context.Context) error {
	_, err := a.query(ctx, "SELECT 1")
	return err
}

// roleName is the pattern a database user name must match. It is strict
// because a name cannot be sent as a query parameter in CREATE ROLE.
var roleName = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

func checkRoleName(role string) error {
	if !roleName.MatchString(role) {
		return fmt.Errorf("%q cannot be used as a database user name; use lowercase letters, digits and underscores, starting with a letter", role)
	}
	return nil
}

// roleSQL builds the statement that creates the user. The name is checked
// against a strict pattern, and any quote in the password is doubled, which
// is how SQL writes a quote inside a quoted value.
func roleSQL(role string, password string) (string, error) {
	if err := checkRoleName(role); err != nil {
		return "", err
	}
	if password == "" {
		return "", errors.New("the database user needs a password")
	}
	escaped := strings.ReplaceAll(password, "'", "''")
	return fmt.Sprintf("CREATE ROLE %s LOGIN CREATEDB PASSWORD '%s'", role, escaped), nil
}

// CreateRole makes the database user Odoo connects as. Odoo creates its own
// databases, so the user is allowed to do that and nothing more.
func (a Admin) CreateRole(ctx context.Context, role string, password string) error {
	sql, err := roleSQL(role, password)
	if err != nil {
		return err
	}
	_, err = a.query(ctx, sql)
	return err
}

// RoleExists asks the administrator whether a database user is already there.
func (a Admin) RoleExists(ctx context.Context, role string) (bool, error) {
	if err := checkRoleName(role); err != nil {
		return false, err
	}
	found, err := a.query(ctx, fmt.Sprintf("SELECT rolname FROM pg_roles WHERE rolname = '%s'", role))
	return len(found) > 0, err
}

// suitable decides whether a login that works is also one Odoo can use.
// answer is what PostgreSQL said to "may this user create databases?".
func suitable(user string, answer string) error {
	if user == "postgres" {
		return errors.New("Odoo refuses to run as the postgres administrator; give a separate user that may create databases")
	}
	if answer != "true" {
		return fmt.Errorf("user %q is not allowed to create databases, which Odoo needs; as the administrator run: ALTER ROLE %s CREATEDB", user, user)
	}
	return nil
}

// Check confirms that the server's login works, and that the user is one
// Odoo can work with.
func Check(ctx context.Context, server Server) error {
	if server.User == "postgres" {
		return suitable(server.User, "")
	}
	answer, err := server.Strings(ctx, "SELECT (rolcreatedb OR rolsuper)::text FROM pg_roles WHERE rolname = current_user")
	if err != nil {
		return err
	}
	if len(answer) == 0 {
		return suitable(server.User, "")
	}
	return suitable(server.User, answer[0])
}
