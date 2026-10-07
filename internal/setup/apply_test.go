package setup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyCreatesFolder(t *testing.T) {
	target := filepath.Join(t.TempDir(), "addons", "custom")
	if err := Apply(folder(target), Request{}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("%s was not created as a folder", target)
	}
}

func TestApplyReportsStepsNotBuiltYet(t *testing.T) {
	err := Apply(file("", t.TempDir(), "Dockerfile"), Request{})

	if !errors.Is(err, ErrNotBuiltYet) {
		t.Fatalf("Apply() error = %v, want ErrNotBuiltYet", err)
	}
}

func TestApplyWritesDevelopmentFiles(t *testing.T) {
	req := Request{Environment: "dev", Version: "19.0", Path: t.TempDir()}
	for _, step := range BuildPlan(req) {
		if step.Kind != KindFile {
			continue
		}
		if err := Apply(step, req); err != nil {
			t.Fatalf("Apply(%s) error = %v", step.Target, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(req.Path, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "FROM odoo:19.0") {
		t.Fatalf("Dockerfile does not use the requested version:\n%s", content)
	}
}

func TestApplyServerPlanHasNoBrokenSteps(t *testing.T) {
	req := Request{Environment: "prod", Version: "19.0", Path: t.TempDir()}
	for _, step := range BuildPlan(req) {
		err := Apply(step, req)
		if err != nil && !errors.Is(err, ErrNotBuiltYet) {
			t.Fatalf("Apply(%s) error = %v", step.Target, err)
		}
	}
}
