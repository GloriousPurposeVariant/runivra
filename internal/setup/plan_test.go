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
	if len(steps) != 7 {
		t.Fatalf("got %d steps, want 7: %v", len(steps), steps)
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

func TestEnterpriseAndCustomSteps(t *testing.T) {
	base := filepath.Join("projects", "shop")

	plain := BuildPlan(Request{Environment: "dev", Version: "19.0", Path: base})
	if plain[1].Kind != KindFolder || plain[2].Kind != KindFolder {
		t.Fatalf("without options both must be empty folders: %v", plain[1:3])
	}

	token := BuildPlan(Request{Environment: "dev", Version: "19.0", Path: base, EnterpriseToken: "secret"})
	if token[1].Kind != KindClone || token[1].Token != "secret" || token[1].Branch != "19.0" {
		t.Fatalf("enterprise step = %+v, want a clone using the token", token[1])
	}

	local := BuildPlan(Request{Environment: "dev", Version: "19.0", Path: base, EnterprisePath: "somewhere"})
	if local[1].Kind != KindCopy || local[1].Source != "somewhere" {
		t.Fatalf("enterprise step = %+v, want a copy", local[1])
	}

	custom := BuildPlan(Request{Environment: "dev", Version: "19.0", Path: base, CustomRepo: "https://example.com/addons.git"})
	if custom[2].Kind != KindClone || custom[2].Branch != "19.0" || custom[2].Shallow {
		t.Fatalf("custom step = %+v, want a full clone of the version branch", custom[2])
	}
}
