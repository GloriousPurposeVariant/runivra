// Package addons works out which folders Odoo loads modules from, where they
// live on this computer, and what kind each one is.
package addons

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GloriousPurposeVariant/runivra/internal/compose"
)

const (
	KindCustom     = "custom"
	KindEnterprise = "enterprise"
	KindImage      = "image"
	KindCore       = "core" // Odoo's own addons, in a source checkout on this computer
)

// Folder is one entry of Odoo's addons_path.
type Folder struct {
	Container string // the path Odoo sees inside the container
	Host      string // the same folder on this computer; "" when it only exists inside the image
	Kind      string
}

// Resolve takes addons_path from odoo.conf and the volumes of the Odoo
// service, and returns one Folder per entry, in the order Odoo lists them.
func Resolve(addonsPath string, volumes []compose.Volume) []Folder {
	var folders []Folder
	for _, entry := range strings.Split(addonsPath, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		folder := Folder{Container: entry, Host: hostFor(entry, volumes), Kind: KindImage}
		if folder.Host != "" {
			folder.Kind = KindCustom
			if looksLikeEnterprise(folder.Host) {
				folder.Kind = KindEnterprise
			}
		}
		folders = append(folders, folder)
	}
	return folders
}

// ResolveHost is Resolve for Odoo running on this computer, where the
// entries of addons_path are already real folders. Relative entries are taken
// from root. Odoo's own addons inside the source checkout are marked core.
func ResolveHost(addonsPath string, root string) []Folder {
	core := map[string]bool{
		filepath.Clean(filepath.Join(root, "addons")):         true,
		filepath.Clean(filepath.Join(root, "odoo", "addons")): true,
	}
	var folders []Folder
	for _, entry := range strings.Split(addonsPath, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		host := filepath.FromSlash(entry)
		if !filepath.IsAbs(host) {
			host = filepath.Join(root, host)
		}
		host = filepath.Clean(host)
		folder := Folder{Container: entry, Host: host, Kind: KindCustom}
		switch {
		case core[host]:
			folder.Kind = KindCore
		case looksLikeEnterprise(host):
			folder.Kind = KindEnterprise
		}
		folders = append(folders, folder)
	}
	return folders
}

// hostFor finds the folder on this computer that a container path comes from.
// The container path may be the mount point itself or a folder below it.
func hostFor(containerPath string, volumes []compose.Volume) string {
	containerPath = path.Clean(containerPath)
	for _, volume := range volumes {
		if volume.Type != "bind" {
			continue
		}
		target := path.Clean(volume.Target)
		if containerPath == target {
			return volume.Source
		}
		if strings.HasPrefix(containerPath, target+"/") {
			return filepath.Join(volume.Source, filepath.FromSlash(strings.TrimPrefix(containerPath, target+"/")))
		}
	}
	return ""
}

// Custom returns only the folders that hold the project's own modules.
func Custom(folders []Folder) []Folder {
	var result []Folder
	for _, folder := range folders {
		if folder.Kind == KindCustom {
			result = append(result, folder)
		}
	}
	return result
}

// Key identifies a set of custom folders. Runivra remembers a database per
// key, so switching to other addons does not reuse the previous database.
func Key(root string, folders []Folder) string {
	var names []string
	for _, folder := range Custom(folders) {
		names = append(names, Relative(root, folder.Host))
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// Relative returns folder relative to root with forward slashes, or the full
// path when it lies outside root.
func Relative(root string, folder string) string {
	relative, err := filepath.Rel(root, folder)
	if err != nil || strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(folder)
	}
	return filepath.ToSlash(relative)
}

// RequirementProblems reports custom folders whose requirements.txt will not
// be installed when the image is built. It returns nothing for a service that
// uses a ready-made image, because then there is no Dockerfile to compare.
func RequirementProblems(service compose.Service, folders []Folder) []string {
	if service.Build == nil {
		return nil
	}
	name := service.Build.Dockerfile
	if name == "" {
		name = "Dockerfile"
	}
	if !filepath.IsAbs(name) {
		name = filepath.Join(service.Build.Context, name)
	}
	dockerfile, err := os.ReadFile(name)
	if err != nil {
		return nil
	}
	ignore, _ := os.ReadFile(filepath.Join(service.Build.Context, ".dockerignore"))

	var problems []string
	for _, folder := range Custom(folders) {
		if _, err := os.Stat(filepath.Join(folder.Host, "requirements.txt")); err != nil {
			continue
		}
		requirements := Relative(service.Build.Context, folder.Host) + "/requirements.txt"
		switch {
		case !strings.Contains(string(dockerfile), requirements):
			problems = append(problems, requirements+" is not installed by the Dockerfile, so its Python packages will be missing in the container.")
		case ignoresEverythingBut(string(ignore), requirements):
			problems = append(problems, requirements+" is left out of the build by .dockerignore, so its Python packages will be missing in the container.")
		}
	}
	return problems
}

// ignoresEverythingBut is true for a .dockerignore that excludes all files
// and does not make an exception for the given one.
func ignoresEverythingBut(ignore string, file string) bool {
	excludesAll, allowed := false, false
	for _, line := range strings.Split(ignore, "\n") {
		line = strings.TrimSpace(line)
		if line == "*" || line == "**" {
			excludesAll = true
		}
		if line == "!"+file {
			allowed = true
		}
	}
	return excludesAll && !allowed
}

// looksLikeEnterprise recognises the Enterprise folder by its content, or by
// its name while it is still empty.
func looksLikeEnterprise(folder string) bool {
	if isDir(filepath.Join(folder, "web_enterprise")) {
		return true
	}
	return strings.Contains(strings.ToLower(filepath.Base(folder)), "enterprise")
}

func isDir(name string) bool {
	info, err := os.Stat(name)
	return err == nil && info.IsDir()
}
