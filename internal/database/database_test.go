package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/postgres"
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

	server := ConnectionFor(view).Server
	want := postgres.Server{Host: "127.0.0.1", Port: 5433, User: "shop", Password: "s3cret"}
	if server != want {
		t.Fatalf("server = %+v; the container's name for this computer must become 127.0.0.1", server)
	}

	view.Config.Docker.DatabaseService = "db"
	args := ConnectionFor(view).command("SELECT 1")
	if args[0] != "docker" || !strings.Contains(strings.Join(args, " "), "exec -T db psql -U shop") {
		t.Fatalf("docker command = %v; want psql run inside the database container", args)
	}
	for _, arg := range args {
		if strings.Contains(arg, "s3cret") {
			t.Fatalf("the password must not be on the command line: %v", args)
		}
	}
}

// A managed database such as AWS RDS: a real host name, and SSL required.
func TestConnectionForRemoteServer(t *testing.T) {
	root := t.TempDir()
	conf := filepath.Join(root, "odoo.conf")
	content := "[options]\ndb_host = shop.abc123.eu-west-1.rds.amazonaws.com\ndb_user = shop\ndb_password = s3cret\ndb_sslmode = require\n"
	if err := os.WriteFile(conf, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	server := ConnectionFor(workspace.View{Root: root, ConfPath: conf}).Server
	if server.Host != "shop.abc123.eu-west-1.rds.amazonaws.com" || server.Port != 5432 || server.SSLMode != "require" {
		t.Fatalf("server = %+v; want the host as written, the default port and SSL required", server)
	}
}

func TestConnectionDefaults(t *testing.T) {
	server := ConnectionFor(workspace.View{Root: t.TempDir()}).Server
	if server.Host != "127.0.0.1" || server.Port != 5432 || server.User != "odoo" || server.SSLMode != "" {
		t.Fatalf("server = %+v; want Odoo's own defaults when odoo.conf says nothing", server)
	}
}

