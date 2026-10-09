package setup

import (
	"context"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/project"
)

func TestDevelopmentSetupSavesTheProject(t *testing.T) {
	req := Request{Environment: "dev", Version: "20.0", Path: t.TempDir(), Name: "shop", Port: 8169, EnterpriseToken: "secret"}
	steps := BuildPlan(req)
	last := steps[len(steps)-1]
	if last.Kind != KindProject {
		t.Fatalf("last step = %+v, want the project to be saved last", last)
	}
	if err := Apply(context.Background(), last, req, nil); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	config, err := project.Load(req.Folder())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Name != "shop" || config.Odoo.Version != "20.0" || config.Odoo.Edition != "enterprise" {
		t.Fatalf("config = %+v", config)
	}
	if config.Docker.OdooService != "web" || config.Paths.Custom != "custom" {
		t.Fatalf("config = %+v", config)
	}
}
