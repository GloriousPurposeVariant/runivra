package setup

import "path/filepath"

const (
	KindFolder = "folder"
	KindFile   = "file"
	KindClone  = "clone"
	KindCopy   = "copy"
)

const (
	OdooRepo       = "https://github.com/odoo/odoo.git"
	EnterpriseRepo = "https://github.com/odoo/enterprise.git"
)

type Step struct {
	Kind     string
	Action   string
	Target   string
	Template string
	URL      string
	Branch   string
	Shallow  bool
	Token    string
	Source   string
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

func enterpriseStep(req Request, target string) Step {
	switch {
	case req.EnterpriseToken != "":
		return Step{Kind: KindClone, Action: "clone Enterprise", Target: target,
			URL: EnterpriseRepo, Branch: req.Version, Shallow: true, Token: req.EnterpriseToken}
	case req.EnterprisePath != "":
		return Step{Kind: KindCopy, Action: "copy Enterprise", Target: target, Source: req.EnterprisePath}
	default:
		return folder(target)
	}
}

func customStep(req Request, target string) Step {
	if req.CustomRepo == "" {
		return folder(target)
	}
	branch := req.CustomBranch
	if branch == "" {
		branch = req.Version
	}
	return Step{Kind: KindClone, Action: "clone custom", Target: target,
		URL: req.CustomRepo, Branch: branch, Token: req.CustomToken}
}

func developmentSteps(req Request) []Step {
	base := req.Folder()
	return []Step{
		{Kind: KindClone, Action: "clone Odoo " + req.Version, Target: base, URL: OdooRepo, Branch: req.Version, Shallow: true},
		enterpriseStep(req, filepath.Join(base, "enterprise")),
		customStep(req, filepath.Join(base, "custom")),
		file("development/odoo.conf", base, "odoo.conf"),
		file("development/Dockerfile", base, "Dockerfile"),
		file("development/docker-compose.yml", base, "docker-compose.yml"),
		file("development/dockerignore", base, ".dockerignore"),
	}
}

func serverSteps(req Request) []Step {
	base := req.Folder()
	return []Step{
		enterpriseStep(req, filepath.Join(base, "addons", "enterprise")),
		customStep(req, filepath.Join(base, "addons", "custom")),
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
