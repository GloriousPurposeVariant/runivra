package setup

import "path/filepath"

type Step struct {
	Action string
	Target string
}

func BuildPlan(req Request) []Step {
	if req.IsDevelopment() {
		return developmentSteps(req)
	}

	steps := serverSteps(req)
	if req.IsProduction() {
		steps = append(steps, productionSteps(req)...)
	}
	return steps
}

func developmentSteps(req Request) []Step {
	base := req.Folder()
	return []Step{
		{Action: "clone Odoo " + req.Version, Target: base},
		{Action: "create folder", Target: filepath.Join(base, "enterprise")},
		{Action: "create folder", Target: filepath.Join(base, "custom")},
	}
}

func serverSteps(req Request) []Step {
	base := req.Folder()
	return []Step{
		{Action: "create folder", Target: filepath.Join(base, "addons", "enterprise")},
		{Action: "create folder", Target: filepath.Join(base, "addons", "custom")},
		{Action: "write file", Target: filepath.Join(base, "config", "odoo.conf")},
		{Action: "write file", Target: filepath.Join(base, "docker-compose.yml")},
		{Action: "write file", Target: filepath.Join(base, "Dockerfile")},
		{Action: "create folder", Target: filepath.Join(base, "logs")},
		{Action: "create folder", Target: filepath.Join(base, "odoo_data")},
	}
}

func productionSteps(req Request) []Step {
	base := req.Folder()
	return []Step{
		{Action: "create folder", Target: filepath.Join(base, "backups")},
	}
}
