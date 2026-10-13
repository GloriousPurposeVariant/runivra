package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var SupportedVersions = []string{"20.0", "19.0", "18.0", "17.0", "16.0"}

type Request struct {
	Environment string
	Version     string
	Path        string
	Name        string

	EnterpriseToken string
	EnterprisePath  string

	CustomRepo   string
	CustomBranch string
	CustomToken  string

	Port int

	// Stack says where the pieces run. Empty means StackDocker.
	Stack      string
	DBUser     string
	DBPassword string
	DBPort     int

	// PythonPath is the Python to build the environment from. It is only
	// used by StackHost.
	PythonPath string
}

const (
	StackDocker = "docker"  // Odoo and PostgreSQL both in Docker
	StackHostDB = "host-db" // Odoo in Docker, PostgreSQL on this computer
	StackHost   = "host"    // Odoo and PostgreSQL both on this computer, no Docker
)

// Database is how Odoo reaches PostgreSQL, as written into odoo.conf.
type Database struct {
	Host     string
	Port     int
	User     string
	Password string
}

// HostDatabase reports whether PostgreSQL runs on this computer and not in a
// container.
func (r Request) HostDatabase() bool {
	return r.Stack == StackHostDB || r.Stack == StackHost
}

// HostOdoo reports whether Odoo itself runs on this computer, without Docker.
func (r Request) HostOdoo() bool {
	return r.Stack == StackHost
}

// HostAddonsPath is addons_path for Odoo running on this computer: real
// folders, in the order Odoo should search them.
func (r Request) HostAddonsPath() string {
	base, err := filepath.Abs(r.Folder())
	if err != nil {
		base = r.Folder()
	}
	return strings.Join([]string{
		filepath.Join(base, "custom"),
		filepath.Join(base, "enterprise"),
		filepath.Join(base, "addons"),
	}, ",")
}

// Database returns the connection settings for the chosen stack. From inside
// a container, this computer is reached under the name host.docker.internal.
func (r Request) Database() Database {
	if !r.HostDatabase() {
		return Database{Host: "db", Port: 5432, User: "odoo", Password: "odoo"}
	}
	port := r.DBPort
	if port == 0 {
		port = 5432
	}
	host := "host.docker.internal"
	if r.HostOdoo() {
		host = "localhost"
	}
	return Database{Host: host, Port: port, User: r.DBUser, Password: r.DBPassword}
}

func (r Request) Problems() []string {
	var problems []string

	switch r.Environment {
	case "":
		problems = append(problems, "--env is required: dev, staging, production")
	case "dev", "development", "stage", "staging", "prod", "production":
	default:
		problems = append(problems, fmt.Sprintf("--env %q is not valid: use dev, staging or prod", r.Environment))
	}

	if r.Version == "" {
		problems = append(problems, "--version is required, for example 19.0")
	} else if !slices.Contains(SupportedVersions, r.Version) {
		problems = append(problems, fmt.Sprintf("--version %q is not supported: use one of %s", r.Version, strings.Join(SupportedVersions, ", ")))
	}

	if strings.ContainsAny(r.Name, `/\`) {
		problems = append(problems, "--name must be a folder name, not a path; put the location in --path")
	}

	if r.EnterpriseToken != "" && r.EnterprisePath != "" {
		problems = append(problems, "give --enterprise-token or --enterprise-path, not both")
	}
	if r.CustomRepo == "" && (r.CustomBranch != "" || r.CustomToken != "") {
		problems = append(problems, "--custom-branch and --custom-token need --custom-repo")
	}

	switch r.Stack {
	case "", StackDocker:
	case StackHostDB, StackHost:
		if !r.IsDevelopment() {
			problems = append(problems, fmt.Sprintf("--stack %s is only available for development", r.Stack))
		}
	default:
		problems = append(problems, fmt.Sprintf("--stack %q is not valid: use %s, %s or %s", r.Stack, StackDocker, StackHostDB, StackHost))

	}

	return problems
}

func (r Request) IsDevelopment() bool {
	return r.Environment == "dev" || r.Environment == "development"
}

func (r Request) IsProduction() bool {
	return r.Environment == "prod" || r.Environment == "production"
}

func (r Request) Folder() string {
	if r.Name == "" {
		return r.Path
	}
	return filepath.Join(r.Path, r.Name)
}

func CheckFolder(folder string) error {
	entries, err := os.ReadDir(folder)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s is not empty; choose another --path, or add --name to create a new folder inside it", folder)
	}
	return nil
}

func CheckSources(req Request) error {
	if req.EnterprisePath == "" {
		return nil
	}
	info, err := os.Stat(req.EnterprisePath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("--enterprise-path %s is not a folder", req.EnterprisePath)
	}
	return nil
}

// EnvironmentFolder is where the Python environment of a host project lives.
func EnvironmentFolder(req Request) string {
	return filepath.Join(req.Folder(), ".venv")
}
