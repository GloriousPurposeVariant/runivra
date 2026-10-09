// Package workspace gives a live picture of a project. The config only says
// where to look; everything here is read from the project's own files each
// time, so it cannot go stale.
package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/GloriousPurposeVariant/runivra/internal/addons"
	"github.com/GloriousPurposeVariant/runivra/internal/compose"
	"github.com/GloriousPurposeVariant/runivra/internal/odooconf"
	"github.com/GloriousPurposeVariant/runivra/internal/project"
)

type View struct {
	Root     string
	Config   project.Config
	Compose  compose.File
	ConfPath string // odoo.conf on this computer; "" when it could not be found
	Port     int    // 0 when the Odoo service publishes no web port
	Addons   []addons.Folder
	Problems []string
}

// Load reads the compose file through Docker Compose and builds the view.
func Load(ctx context.Context, root string, config project.Config) (View, error) {
	file, err := compose.Load(ctx, filepath.Join(root, filepath.FromSlash(config.Docker.ComposeFile)))
	if err != nil {
		return View{Root: root, Config: config}, err
	}
	return Build(root, config, file), nil
}

// Build assembles the view from compose data that has already been read.
func Build(root string, config project.Config, file compose.File) View {
	view := View{Root: root, Config: config, Compose: file}
	odoo := file.Services[config.Docker.OdooService]

	view.ConfPath = confPath(root, config, odoo)
	if port, err := file.PublishedPort(config.Docker.OdooService, 8069); err == nil {
		view.Port = port
	}

	if view.ConfPath != "" {
		if values, err := odooconf.Read(view.ConfPath); err == nil {
			view.Addons = addons.Resolve(values["addons_path"], odoo.Volumes)
		}
	}
	view.Addons = withOverride(view.Addons, root, config.Paths.Custom, addons.KindCustom)
	view.Addons = withOverride(view.Addons, root, config.Paths.Enterprise, addons.KindEnterprise)

	view.Problems = addons.RequirementProblems(odoo, view.Addons)
	return view
}

// AddonsKey identifies the custom folders in use right now.
func (v View) AddonsKey() string {
	return addons.Key(v.Root, v.Addons)
}

// confPath prefers a path written in the config, then the .conf file mounted
// into the Odoo service, then an odoo.conf at the top of the project.
func confPath(root string, config project.Config, odoo compose.Service) string {
	if config.Paths.Config != "" {
		return absolute(root, config.Paths.Config)
	}
	for _, volume := range odoo.Volumes {
		if volume.Type == "bind" && strings.HasSuffix(volume.Target, ".conf") {
			return volume.Source
		}
	}
	fallback := filepath.Join(root, "odoo.conf")
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	return ""
}

// withOverride adds a folder named in the config when detection missed it.
func withOverride(folders []addons.Folder, root string, configured string, kind string) []addons.Folder {
	if configured == "" {
		return folders
	}
	host := absolute(root, configured)
	for _, folder := range folders {
		if folder.Host == host {
			return folders
		}
	}
	return append(folders, addons.Folder{Host: host, Kind: kind})
}

func absolute(root string, name string) string {
	name = filepath.FromSlash(name)
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(root, name)
}
