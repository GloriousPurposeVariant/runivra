package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateIsEmptyUntilSaved(t *testing.T) {
	root := t.TempDir()
	state, err := LoadState(root)
	if err != nil || state.Database("custom") != "" {
		t.Fatalf("state = %+v, error = %v; want an empty state and no error", state, err)
	}

	state.SetDatabase("custom", "shop_dev")
	if err := SaveState(root, state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}
	state, err = LoadState(root)
	if err != nil || state.Database("custom") != "shop_dev" || state.Database("custom_proj") != "" {
		t.Fatalf("state = %+v, error = %v; want shop_dev", state, err)
	}

	ignore, err := os.ReadFile(filepath.Join(root, DirName, ".gitignore"))
	if err != nil || !strings.Contains(string(ignore), StateName) {
		t.Fatalf(".gitignore = %q, error = %v; the state file must be ignored by Git", ignore, err)
	}
}
