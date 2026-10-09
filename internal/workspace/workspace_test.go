package workspace

import (
	"os"
	"path/filepath"
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
