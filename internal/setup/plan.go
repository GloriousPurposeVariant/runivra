package setup

import "path/filepath"

const (
	KindFolder = "folder"
	KindFile   = "file"
	KindClone  = "clone"
)

type Step struct {
	Kind     string
	Action   string
	Target   string
	Template string
}

func folder(parts ...string) Step {
	return Step{Kind: KindFolder, Action: "create folder", Target: filepath.Join(parts...)}
}

func file(template string, parts ...string) Step {
	return Step{Kind: KindFile, Action: "write file", Target: filepath.Join(parts...), Template: template}
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
		{Kind: KindClone, Action: "clone Odoo " + req.Version, Target: base},
		folder(base, "enterprise"),
		folder(base, "custom"),
		file("development/odoo.conf", base, "odoo.conf"),
		file("development/Dockerfile", base, "Dockerfile"),
		file("development/docker-compose.yml", base, "docker-compose.yml"),
		file("development/dockerignore", base, ".dockerignore"),
	}
}

func serverSteps(req Request) []Step {
	base := req.Folder()
	return []Step{
		folder(base, "addons", "enterprise"),
		folder(base, "addons", "custom"),
		file("", base, "config", "odoo.conf"),
		file("", base, "docker-compose.yml"),
		file("", base, "Dockerfile"),
		folder(base, "logs"),
		folder(base, "odoo_data"),
	}
}

func productionSteps(req Request) []Step {
	base := req.Folder()
	return []Step{
		folder(base, "backups"),
	}
}
