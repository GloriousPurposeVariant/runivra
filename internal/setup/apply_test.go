package setup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyCreatesFolder(t *testing.T) {
	target := filepath.Join(t.TempDir(), "addons", "custom")
	if err := Apply(folder(target)); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("%s was not created as a folder", target)
	}
}

func TestApplyReportsStepsNotBuiltYet(t *testing.T) {
	err := Apply(file(t.TempDir(), "Dockerfile"))
	if !errors.Is(err, ErrNotBuiltYet) {
		t.Fatalf("Apply() error = %v, want ErrNotBuiltYet", err)
	}
}
