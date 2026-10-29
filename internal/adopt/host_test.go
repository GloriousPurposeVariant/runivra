package adopt

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/project"
	"github.com/GloriousPurposeVariant/runivra/internal/python"
)

// Odoo cloned and installed by hand: the config file sits in config/, the
// environment is called venv, and Enterprise lives outside the project.
func TestDetectHandInstalledProject(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, "#!/usr/bin/env python3\n", root, "odoo-bin")
	mustMkdir(t, root, "odoo")
	mustWrite(t, "version_info = (17, 0, 0, FINAL, 0, '')\n", root, "odoo", "release.py")
	mustMkdir(t, root, "config")
	enterprise := mustMkdir(t, t.TempDir(), "ee")
	mustMkdir(t, enterprise, "web_enterprise")
	mustMkdir(t, root, "extra")
	mustWrite(t, "[options]\naddons_path = addons,"+enterprise+",extra\n", root, "config", "odoo.conf")
	environment := python.EnvironmentPython(runtime.GOOS, filepath.Join(root, "venv"))
	mustMkdir(t, filepath.Dir(environment))
	mustWrite(t, "", environment)

	if !IsOdooSource(root) {
		t.Fatal("IsOdooSource() = false for a folder with odoo-bin")
	}
	config := DetectHost(root)
	if config.Odoo != (project.Odoo{Version: "17.0", Edition: "enterprise", Runtime: "host"}) {
		t.Fatalf("odoo = %+v, want 17.0 enterprise host", config.Odoo)
	}
	want := project.Paths{Config: "config/odoo.conf", Custom: "extra", Enterprise: filepath.ToSlash(enterprise), Python: "venv"}
	if config.Paths != want {
		t.Fatalf("paths = %+v", config.Paths)
	}
}

// A bare checkout with nothing set up yet.
func TestDetectBareCheckout(t *testing.T) {
	root := t.TempDir()
	if IsOdooSource(root) {
		t.Fatal("IsOdooSource() = true for an empty folder")
	}
	mustWrite(t, "", root, "odoo-bin")
	config := DetectHost(root)
	if config.Odoo.Version != "" || config.Odoo.Edition != "community" || config.Paths.Config != "" || config.Paths.Python != "" {
		t.Fatalf("config = %+v", config)
	}
}
