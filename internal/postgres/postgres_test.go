package postgres

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func only(names ...string) func(string) bool {
	return func(name string) bool {
		for _, candidate := range names {
			if candidate == name {
				return true
			}
		}
		return false
	}
}

func TestInstallCommands(t *testing.T) {
	windows, err := installCommands("windows", false, only("winget"), "secret1")
	if err != nil || len(windows) != 1 || windows[0][0] != "winget" {
		t.Fatalf("windows = %v, %v", windows, err)
	}
	override := windows[0][len(windows[0])-1]
	if !strings.Contains(override, "--mode unattended") || !strings.Contains(override, "--superpassword secret1") {
		t.Fatalf("override = %q, want an unattended install with the administrator password", override)
	}

	mac, err := installCommands("darwin", false, only("brew"), "")
	if err != nil || len(mac) != 2 || mac[1][1] != "services" {
		t.Fatalf("darwin = %v, %v; want install then start", mac, err)
	}

	debian, err := installCommands("linux", false, only("apt-get"), "")
	if err != nil || debian[1][0] != "sudo" || debian[1][1] != "apt-get" {
		t.Fatalf("debian as a normal user = %v, %v; want sudo apt-get", debian, err)
	}

	asRoot, err := installCommands("linux", true, only("apt-get"), "")
	if err != nil || asRoot[1][0] != "apt-get" {
		t.Fatalf("debian as root = %v, %v; want no sudo", asRoot, err)
	}

	fedora, err := installCommands("linux", false, only("dnf"), "")
	if err != nil || len(fedora) != 3 {
		t.Fatalf("fedora = %v, %v; want install, initialise and start", fedora, err)
	}

	if _, err := installCommands("windows", false, only(), ""); err == nil {
		t.Fatal("Windows without winget must return an error")
	}
}

func TestAdminCommand(t *testing.T) {
	args, env := adminCommand("windows", false, `C:\pg\psql.exe`, "secret1", "SELECT 1")
	if args[0] != `C:\pg\psql.exe` || len(env) != 1 || env[0] != "PGPASSWORD=secret1" {
		t.Fatalf("windows = %v, %v; want psql with the password in the environment", args, env)
	}
	for _, arg := range args {
		if strings.Contains(arg, "secret1") {
			t.Fatalf("the administrator password must not be on the command line: %v", args)
		}
	}

	args, env = adminCommand("linux", false, "psql", "", "SELECT 1")
	if args[0] != "sudo" || args[2] != "postgres" || env != nil {
		t.Fatalf("linux = %v, %v; want sudo -u postgres", args, env)
	}

	args, _ = adminCommand("linux", true, "psql", "", "SELECT 1")
	if args[0] != "runuser" {
		t.Fatalf("linux as root = %v; want runuser", args)
	}

	args, _ = adminCommand("darwin", false, "psql", "", "SELECT 1")
	if args[0] != "psql" {
		t.Fatalf("darwin = %v; want psql run directly", args)
	}
}

func TestNewPassword(t *testing.T) {
	first, second := NewPassword(), NewPassword()
	if len(first) != 24 || first == second {
		t.Fatalf("passwords = %q, %q; want 24 characters and different each time", first, second)
	}
	if strings.ContainsAny(first, " '\"\\$") {
		t.Fatalf("password %q contains a character that is unsafe on a command line", first)
	}
}

func TestPortOpen(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if !portOpen(port) {
		t.Fatalf("port %d has a listener but was reported closed", port)
	}
	listener.Close()
	if portOpen(port) {
		t.Fatalf("port %d was closed but is still reported open", port)
	}
}

func TestOnPath(t *testing.T) {
	bin := filepath.Join("pg", "17", "bin")
	entries := []string{"tools", bin + string(os.PathSeparator), "other"}
	value := strings.Join(entries, string(os.PathListSeparator))

	if !onPath(value, strings.ToUpper(bin)) {
		t.Fatal("the folder is on the path, ignoring capitals and the trailing separator")
	}
	if onPath(value, filepath.Join("pg", "16", "bin")) {
		t.Fatal("a different version's folder is not on the path")
	}
	if onPath("", "tools") {
		t.Fatal("an empty path contains nothing")
	}
}

func TestRoleSQL(t *testing.T) {
	sql, err := roleSQL("odoo", "it's a secret")
	if err != nil || sql != "CREATE ROLE odoo LOGIN CREATEDB PASSWORD 'it''s a secret'" {
		t.Fatalf("sql = %q, error = %v; a quote in the password must be doubled", sql, err)
	}
	for _, bad := range []string{"", "Odoo", "1odoo", "odoo; DROP ROLE postgres", "my user"} {
		if _, err := roleSQL(bad, "x"); err == nil {
			t.Fatalf("user name %q must be refused", bad)
		}
	}
	if _, err := roleSQL("odoo", ""); err == nil {
		t.Fatal("an empty password must be refused")
	}
}

func TestSuitable(t *testing.T) {
	if err := suitable("odoo", "t\n"); err != nil {
		t.Fatalf("a user that may create databases is fine: %v", err)
	}
	if err := suitable("odoo", "f\n"); err == nil {
		t.Fatal("a user that may not create databases must be refused")
	}
	if err := suitable("postgres", "t\n"); err == nil {
		t.Fatal("the postgres administrator must be refused, because Odoo will not run as it")
	}
}
