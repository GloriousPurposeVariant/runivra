package setup

import (
	"path/filepath"
	"testing"
)

func hasTarget(steps []Step, target string) bool {
	for _, step := range steps {
		if step.Target == target {
			return true
		}
	}
	return false
}

func TestBuildPlanForDevelopment(t *testing.T) {
	base := filepath.Join("projects", "shop")
	steps := BuildPlan(Request{Environment: "dev", Version: "19.0", Path: base})
	if len(steps) != 3 {
		t.Fatalf("got %d steps, want 3: %v", len(steps), steps)
	}
	if steps[0].Action != "clone Odoo 19.0" || steps[0].Target != base {
		t.Fatalf("first step = %v, want Odoo cloned into the project folder", steps[0])
	}
	if !hasTarget(steps, filepath.Join(base, "enterprise")) {
		t.Fatalf("development plan has no enterprise folder: %v", steps)
	}
}

func TestBuildPlanForStagingHasNoBackups(t *testing.T) {
	base := filepath.Join("projects", "shop")
	steps := BuildPlan(Request{Environment: "staging", Version: "19.0", Path: base})
	if len(steps) != 7 {
		t.Fatalf("got %d steps, want 7: %v", len(steps), steps)
	}
	if hasTarget(steps, filepath.Join(base, "backups")) {
		t.Fatalf("staging must not get a backups folder: %v", steps)
	}
}

func TestBuildPlanForProductionAddsBackups(t *testing.T) {
	base := filepath.Join("projects", "shop")
	steps := BuildPlan(Request{Environment: "prod", Version: "19.0", Path: base})
	if len(steps) != 8 {
		t.Fatalf("got %d steps, want 8: %v", len(steps), steps)
	}
	if !hasTarget(steps, filepath.Join(base, "backups")) {
		t.Fatalf("production plan has no backups folder: %v", steps)
	}
}
