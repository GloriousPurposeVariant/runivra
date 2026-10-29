package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/addons"
	"github.com/GloriousPurposeVariant/runivra/internal/compose"
	"github.com/GloriousPurposeVariant/runivra/internal/project"
)

func TestBuildReadsTheProjectLive(t *testing.T) {
	root := t.TempDir()
	custom := filepath.Join(root, "custom_proj")
	enterprise := filepath.Join(root, "enterprise")
	for _, dir := range []string{custom, filepath.Join(enterprise, "web_enterprise")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	conf := filepath.Join(root, "odoo.conf")
	if err := os.WriteFile(conf, []byte("[options]\naddons_path = /opt/odoo/custom,/opt/odoo/enterprise\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	file := compose.File{Services: map[string]compose.Service{
		"web": {
			Image: "odoo:20.0",
			Ports: []compose.Port{{Target: 8069, Published: "8555"}},
			Volumes: []compose.Volume{
				{Type: "bind", Source: custom, Target: "/opt/odoo/custom"},
				{Type: "bind", Source: enterprise, Target: "/opt/odoo/enterprise"},
				{Type: "bind", Source: conf, Target: "/etc/odoo/odoo.conf"},
			},
		},
	}}
	config := project.Config{Name: "shop", Docker: project.Docker{ComposeFile: "docker-compose.yml", OdooService: "web"}}

	view := Build(root, config, file)
	if view.ConfPath != conf || view.Port != 8555 {
		t.Fatalf("conf = %q, port = %d; want the mounted conf and the published port", view.ConfPath, view.Port)
	}
	if len(view.Addons) != 2 || view.Addons[0].Host != custom || view.Addons[0].Kind != addons.KindCustom || view.Addons[1].Kind != addons.KindEnterprise {
		t.Fatalf("addons = %+v", view.Addons)
	}
	if view.AddonsKey() != "custom_proj" {
		t.Fatalf("AddonsKey() = %q, want custom_proj", view.AddonsKey())
	}
}

func TestBuildHostReadsOdooConf(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "custom"), 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "[options]\naddons_path = " + filepath.Join(root, "custom") + "," + filepath.Join(root, "addons") + "\nhttp_port = 8169\n"
	if err := os.WriteFile(filepath.Join(root, "odoo.conf"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	config := project.Config{Name: "shop", Odoo: project.Odoo{Runtime: "host"}, Paths: project.Paths{Config: "odoo.conf", Python: ".venv"}}

	view := BuildHost(root, config)
	if view.Port != 8169 || view.ConfPath != filepath.Join(root, "odoo.conf") {
		t.Fatalf("port = %d, conf = %q", view.Port, view.ConfPath)
	}
	if len(view.Addons) != 2 || view.Addons[0].Kind != addons.KindCustom || view.Addons[1].Kind != addons.KindCore {
		t.Fatalf("addons = %+v", view.Addons)
	}
	if len(view.Problems) != 1 || !strings.Contains(view.Problems[0], ".venv is missing") {
		t.Fatalf("problems = %q; a missing environment must be reported", view.Problems)
	}
}
