package setup

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func TestApplyClonesBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	source := t.TempDir()
	runGit(t, source, "init", "--quiet", "--initial-branch=19.0")
	if err := os.WriteFile(filepath.Join(source, "odoo-bin"), []byte("#!/usr/bin/env python3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, source, "add", ".")
	runGit(t, source, "-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "first")

	target := filepath.Join(t.TempDir(), "project")
	step := Step{Kind: KindClone, Target: target, URL: source, Branch: "19.0"}
	if err := Apply(context.Background(), step, Request{}, nil); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "odoo-bin")); err != nil {
		t.Fatalf("the cloned folder has no odoo-bin: %v", err)
	}

	missing := Step{Kind: KindClone, Target: filepath.Join(t.TempDir(), "other"), URL: source, Branch: "no-such-branch"}
	if err := Apply(context.Background(), missing, Request{}, nil); err == nil {
		t.Fatal("cloning a branch that does not exist must fail")
	}
}

func TestApplyCreatesFolder(t *testing.T) {
	target := filepath.Join(t.TempDir(), "addons", "custom")
	if err := Apply(context.Background(), folder(target), Request{}, nil); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("%s was not created as a folder", target)
	}
}

func TestApplyReportsStepsNotBuiltYet(t *testing.T) {
	err := Apply(context.Background(), file("", t.TempDir(), "Dockerfile"), Request{}, nil)

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
		if err := Apply(context.Background(), step, req, nil); err != nil {
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
		err := Apply(context.Background(), step, req, nil)
		if err != nil && !errors.Is(err, ErrNotBuiltYet) {
			t.Fatalf("Apply(%s) error = %v", step.Target, err)
		}
	}
}
