package adopt

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/GloriousPurposeVariant/runivra/internal/addons"
	"github.com/GloriousPurposeVariant/runivra/internal/odooconf"
	"github.com/GloriousPurposeVariant/runivra/internal/project"
	"github.com/GloriousPurposeVariant/runivra/internal/python"
)

// Where a hand-installed project usually keeps its config file and its
// Python environment, in the order they are tried.
var (
	confNames        = []string{"odoo.conf", "odoo-server.conf", "config/odoo.conf", "conf/odoo.conf", "etc/odoo.conf"}
	environmentNames = []string{".venv", "venv", "env"}
)

var releaseVersion = regexp.MustCompile(`(?m)^version_info\s*=\s*\(\s*(\d+)`)

// IsOdooSource is true when root holds an Odoo source checkout.
func IsOdooSource(root string) bool {
	return fileExists(filepath.Join(root, "odoo-bin"))
}

// DetectHost fills in the configuration of a project that runs Odoo from
// source on this computer, without Docker.
func DetectHost(root string) project.Config {
	config := project.Config{Name: filepath.Base(root), Environment: "development"}
	config.Odoo.Runtime = "host"
	config.Odoo.Version = sourceVersion(root)
	config.Odoo.Edition = "community"

	for _, name := range confNames {
		if fileExists(filepath.Join(root, filepath.FromSlash(name))) {
			config.Paths.Config = name
			break
		}
	}
	for _, name := range environmentNames {
		if fileExists(python.EnvironmentPython(runtime.GOOS, filepath.Join(root, name))) {
			config.Paths.Python = name
			break
		}
	}

	if config.Paths.Config == "" {
		return config
	}
	values, err := odooconf.Read(filepath.Join(root, filepath.FromSlash(config.Paths.Config)))
	if err != nil {
		return config
	}
	for _, folder := range addons.ResolveHost(values["addons_path"], root) {
		switch {
		case folder.Kind == addons.KindEnterprise && config.Paths.Enterprise == "":
			config.Paths.Enterprise = relativeTo(root, folder.Host)
			if isDir(filepath.Join(folder.Host, "web_enterprise")) {
				config.Odoo.Edition = "enterprise"
			}
		case folder.Kind == addons.KindCustom && config.Paths.Custom == "":
			config.Paths.Custom = relativeTo(root, folder.Host)
		}
	}
	return config
}

// sourceVersion reads the major version from odoo/release.py, for example
// "17.0". It returns "" when the file is missing or unusual.
func sourceVersion(root string) string {
	content, err := os.ReadFile(filepath.Join(root, "odoo", "release.py"))
	if err != nil {
		return ""
	}
	if match := releaseVersion.FindStringSubmatch(string(content)); match != nil {
		return match[1] + ".0"
	}
	return ""
}

func fileExists(name string) bool {
	info, err := os.Stat(name)
	return err == nil && !info.IsDir()
}
