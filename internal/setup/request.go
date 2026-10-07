package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Request struct {
	Environment string
	Version     string
	Path        string
	Name        string
}

func (r Request) Problems() []string {
	var problems []string

	switch r.Environment {
	case "":
		problems = append(problems, "--env is required: dev, staging or prod")
	case "dev", "development", "stage", "staging", "prod", "production":
	default:
		problems = append(problems, fmt.Sprintf("--env %q is not valid: use dev, staging or prod", r.Environment))
	}

	if r.Version == "" {
		problems = append(problems, "--version is required, for example 19.0")
	}

	if strings.ContainsAny(r.Name, `/\`) {
		problems = append(problems, "--name must be a folder name, not a path; put the location in --path")
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
