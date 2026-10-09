package adopt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/compose"
)

func mustMkdir(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(parts...)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustWrite(t *testing.T, content string, parts ...string) string {
	t.Helper()
	path := filepath.Join(parts...)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindComposeFile(t *testing.T) {
	root := t.TempDir()
	if got := FindComposeFile(root); got != "" {
		t.Fatalf("FindComposeFile() = %q in an empty folder, want none", got)
	}
	mustWrite(t, "services: {}\n", root, "compose.yaml")
	if got := FindComposeFile(root); got != "compose.yaml" {
		t.Fatalf("FindComposeFile() = %q, want compose.yaml", got)
	}
}

// A project laid out by hand: Odoo is built from a Dockerfile and sits behind
// Nginx, so it publishes no web port itself, and the addons folder has its
// own name.
func TestDetectHandMadeProject(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, "FROM odoo:18\nUSER root\n", root, "Dockerfile")
	conf := mustWrite(t, "[options]\n", root, "odoo.conf")
	script := mustWrite(t, "#!/bin/sh\n", root, "odoo-upgrade.sh")
	production := mustMkdir(t, root, "production")
	enterprise := mustMkdir(t, root, "enterprise")
	mustMkdir(t, enterprise, "web_enterprise")

	file := compose.File{Services: map[string]compose.Service{
		"nginx": {Image: "nginx:alpine", Ports: []compose.Port{{Target: 80, Published: "8069"}}},
		"db":    {Image: "postgres:15"},
		"web": {
			Build: &compose.Build{Context: root, Dockerfile: "Dockerfile"},
			Ports: []compose.Port{{Target: 8072, Published: "8072"}},
			Volumes: []compose.Volume{
				{Type: "bind", Source: enterprise, Target: "/mnt/extra-addons/enterprise"},
				{Type: "bind", Source: production, Target: "/mnt/extra-addons/custom"},
				{Type: "bind", Source: conf, Target: "/etc/odoo/odoo.conf"},
				{Type: "bind", Source: script, Target: "/usr/local/bin/odoo-upgrade"},
				{Type: "volume", Source: "odoo-data", Target: "/var/lib/odoo"},
			},
		},
	}}

	config := Detect(root, "docker-compose.yml", file)
	if config.Docker.OdooService != "web" || config.Docker.DatabaseService != "db" {
		t.Fatalf("services = %+v", config.Docker)
	}
	if config.Odoo.Version != "18.0" || config.Odoo.Edition != "enterprise" {
		t.Fatalf("odoo = %+v, want 18.0 enterprise", config.Odoo)
	}
	if config.Paths.Config != "odoo.conf" || config.Paths.Custom != "production" || config.Paths.Enterprise != "enterprise" {
		t.Fatalf("paths = %+v", config.Paths)
	}
	if config.Name != filepath.Base(root) || config.Environment != "development" {
		t.Fatalf("name = %q, environment = %q", config.Name, config.Environment)
	}
}

// A Community project using the official image directly, with an empty
// enterprise folder and nothing built.
func TestDetectImageOnlyCommunityProject(t *testing.T) {
	root := t.TempDir()
	custom := mustMkdir(t, root, "addons")
	enterprise := mustMkdir(t, root, "enterprise")

	file := compose.File{Services: map[string]compose.Service{
		"postgres": {Image: "postgres:16"},
		"odoo": {
			Image: "odoo:17.0",
			Ports: []compose.Port{{Target: 8069, Published: "8069"}},
			Volumes: []compose.Volume{
				{Type: "bind", Source: custom, Target: "/mnt/extra-addons"},
				{Type: "bind", Source: enterprise, Target: "/mnt/enterprise"},
			},
		},
	}}

	config := Detect(root, "compose.yaml", file)
	if config.Docker.OdooService != "odoo" || config.Docker.DatabaseService != "postgres" {
		t.Fatalf("services = %+v", config.Docker)
	}
	if config.Odoo.Version != "17.0" || config.Odoo.Edition != "community" {
		t.Fatalf("odoo = %+v, want 17.0 community", config.Odoo)
	}
	if config.Paths.Custom != "addons" || config.Paths.Enterprise != "enterprise" || config.Paths.Config != "" {
		t.Fatalf("paths = %+v", config.Paths)
	}
}
