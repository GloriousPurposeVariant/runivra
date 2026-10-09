// Package adopt works out the configuration of an Odoo project that already
// exists, by reading its compose file and looking at its folders.
package adopt

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GloriousPurposeVariant/runivra/internal/compose"
	"github.com/GloriousPurposeVariant/runivra/internal/project"
)

var composeNames = []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}

var odooImage = regexp.MustCompile(`(?im)^\s*(?:FROM\s+)?odoo:(\d{2})(?:\.\d)?`)

// FindComposeFile returns the name of the compose file in root, or "" when
// there is none.
func FindComposeFile(root string) string {
	for _, name := range composeNames {
		if info, err := os.Stat(filepath.Join(root, name)); err == nil && !info.IsDir() {
			return name
		}
	}
	return ""
}

// Detect fills in as much of the configuration as the project reveals. Values
// it cannot work out are left empty for the user to supply.
func Detect(root string, composeName string, file compose.File) project.Config {
	config := project.Config{Name: filepath.Base(root), Environment: "development"}
	config.Docker.ComposeFile = composeName
	config.Docker.OdooService = findOdooService(file)
	config.Docker.DatabaseService = findDatabaseService(file)

	odoo := file.Services[config.Docker.OdooService]
	config.Odoo.Version = versionOf(odoo)
	config.Odoo.Edition = "community"

	for _, volume := range odoo.Volumes {
		if volume.Type != "bind" {
			continue
		}
		relative := relativeTo(root, volume.Source)
		switch {
		case strings.HasSuffix(volume.Target, ".conf"):
			config.Paths.Config = relative
		case !isDir(volume.Source):
		case looksLikeEnterprise(volume.Source):
			config.Paths.Enterprise = relative
			if isDir(filepath.Join(volume.Source, "web_enterprise")) {
				config.Odoo.Edition = "enterprise"
			}
		case config.Paths.Custom == "" || strings.Contains(volume.Target, "custom"):
			config.Paths.Custom = relative
		}
	}
	return config
}

func serviceNames(file compose.File) []string {
	names := make([]string, 0, len(file.Services))
	for name := range file.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func findOdooService(file compose.File) string {
	names := serviceNames(file)
	for _, name := range names {
		if versionOf(file.Services[name]) != "" {
			return name
		}
	}
	for _, name := range names {
		for _, port := range file.Services[name].Ports {
			if port.Target == 8069 {
				return name
			}
		}
	}
	for _, name := range names {
		if name == "web" || name == "odoo" {
			return name
		}
	}
	return ""
}

func findDatabaseService(file compose.File) string {
	names := serviceNames(file)
	for _, name := range names {
		if strings.Contains(file.Services[name].Image, "postgres") {
			return name
		}
	}
	for _, name := range names {
		if name == "db" || name == "postgres" {
			return name
		}
	}
	return ""
}

// versionOf reads the Odoo version from a service's image, or from the
// Dockerfile it is built from. It returns "" for a service that is not Odoo.
func versionOf(service compose.Service) string {
	if match := odooImage.FindStringSubmatch(service.Image); match != nil {
		return match[1] + ".0"
	}
	if service.Build == nil {
		return ""
	}
	name := service.Build.Dockerfile
	if name == "" {
		name = "Dockerfile"
	}
	if !filepath.IsAbs(name) {
		name = filepath.Join(service.Build.Context, name)
	}
	content, err := os.ReadFile(name)
	if err != nil {
		return ""
	}
	if match := odooImage.FindStringSubmatch(string(content)); match != nil {
		return match[1] + ".0"
	}
	return ""
}

func looksLikeEnterprise(folder string) bool {
	if isDir(filepath.Join(folder, "web_enterprise")) {
		return true
	}
	return strings.Contains(strings.ToLower(filepath.Base(folder)), "enterprise")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// relativeTo returns path relative to root with forward slashes, or the full
// path when it lies outside the project.
func relativeTo(root string, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}
