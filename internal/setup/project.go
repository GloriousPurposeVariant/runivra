package setup

import (
	"path/filepath"

	"github.com/GloriousPurposeVariant/runivra/internal/project"
)

func projectConfig(req Request) project.Config {
	name := req.Name
	if name == "" {
		if absolute, err := filepath.Abs(req.Folder()); err == nil {
			name = filepath.Base(absolute)
		}
	}

	edition := "community"
	if req.EnterpriseToken != "" || req.EnterprisePath != "" {
		edition = "enterprise"
	}

	return project.Config{
		Name:        name,
		Environment: "development",
		Odoo:        project.Odoo{Version: req.Version, Edition: edition},
		Paths:       project.Paths{Config: "odoo.conf", Custom: "custom", Enterprise: "enterprise"},
		Docker:      project.Docker{ComposeFile: "docker-compose.yml", OdooService: "web", DatabaseService: "db"},
	}
}
