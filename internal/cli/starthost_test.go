package cli

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

func TestHostCommandRunsOdooFromTheEnvironment(t *testing.T) {
	req := setup.Request{Path: filepath.Join("projects", "shop"), Stack: setup.StackHost}
	command := hostCommand(req)
	if len(command) != 4 || command[2] != "--config" {
		t.Fatalf("command = %v", command)
	}
	if filepath.Dir(filepath.Dir(command[0])) != filepath.Join("projects", "shop", ".venv") {
		t.Fatalf("python = %q, want the one inside the project's environment", command[0])
	}
	if command[1] != filepath.Join("projects", "shop", "odoo-bin") || command[3] != filepath.Join("projects", "shop", "odoo.conf") {
		t.Fatalf("command = %v", command)
	}
}

func TestWarningGetsItsOwnMark(t *testing.T) {
	if resultMark(&setup.Warning{Summary: "x"}) != markWarn {
		t.Fatal("a step that finished with a warning must not be marked done or failed")
	}
	if resultMark(errors.New("boom")) != markFail {
		t.Fatal("an ordinary error must still be marked failed")
	}
}
