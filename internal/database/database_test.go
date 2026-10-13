package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/workspace"
)

func TestParseNames(t *testing.T) {
	names := parseNames("shop_dev\r\nshop_test\n\n")
	if len(names) != 2 || names[0] != "shop_dev" || names[1] != "shop_test" {
		t.Fatalf("names = %q", names)
	}
	if len(parseNames("")) != 0 {
		t.Fatal("no output must mean no databases")
	}
}

func TestChoose(t *testing.T) {
	names := []string{"shop_dev", "shop_test"}

	if got, err := Choose(names, "shop_test", "shop_dev"); err != nil || got != "shop_test" {
		t.Fatalf("an explicit name must win: got %q, %v", got, err)
	}
	if _, err := Choose(names, "missing", ""); err == nil {
		t.Fatal("an explicit name that does not exist must be an error")
	}
	if got, _ := Choose(names, "", "shop_dev"); got != "shop_dev" {
		t.Fatalf("the remembered database must be used: got %q", got)
	}
	if got, _ := Choose(names, "", "dropped_since"); got != "" {
		t.Fatalf("a remembered database that no longer exists must not be used: got %q", got)
	}
	if got, _ := Choose([]string{"only"}, "", ""); got != "only" {
		t.Fatalf("a single database must be chosen without asking: got %q", got)
	}
	if got, err := Choose(names, "", ""); got != "" || err != nil {
		t.Fatalf("with several and nothing remembered the caller must ask: got %q, %v", got, err)
	}
}

func TestConnectionForReadsOdooConf(t *testing.T) {
	root := t.TempDir()
	conf := filepath.Join(root, "odoo.conf")
	content := "[options]\ndb_host = host.docker.internal\ndb_port = 5433\ndb_user = shop\ndb_password = s3cret\n"
	if err := os.WriteFile(conf, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	view := workspace.View{Root: root, ConfPath: conf}
	view.Config.Docker.ComposeFile = "docker-compose.yml"

	host := ConnectionFor(view)
	if host.Host != "127.0.0.1" || host.Port != "5433" || host.User != "shop" || host.Password != "s3cret" {
		t.Fatalf("connection = %+v; the container's name for this computer must become 127.0.0.1", host)
	}
	args, env := host.command("psql", "SELECT 1")
	if args[0] != "psql" || len(env) != 1 || env[0] != "PGPASSWORD=s3cret" {
		t.Fatalf("host command = %v, %v; want psql run here with the password in the environment", args, env)
	}
	for _, arg := range args {
		if strings.Contains(arg, "s3cret") {
			t.Fatalf("the password must not be on the command line: %v", args)
		}
	}

	view.Config.Docker.DatabaseService = "db"
	args, env = ConnectionFor(view).command("", "SELECT 1")
	if args[0] != "docker" || env != nil {
		t.Fatalf("docker command = %v, %v; want psql run inside the database container", args, env)
	}
}

func TestConnectionDefaults(t *testing.T) {
	connection := ConnectionFor(workspace.View{Root: t.TempDir()})
	if connection.Host != "127.0.0.1" || connection.Port != "5432" || connection.User != "odoo" {
		t.Fatalf("connection = %+v; want Odoo's own defaults when odoo.conf says nothing", connection)
	}
}
