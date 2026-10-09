package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DirName       = ".runivra"
	ConfigName    = "config.yml"
	SchemaVersion = 1
)

var ErrNotFound = errors.New("no Runivra project found in this folder or any folder above it")

type Config struct {
	Schema      int    `yaml:"schema"`
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Odoo        Odoo   `yaml:"odoo"`
	Paths       Paths  `yaml:"paths,omitempty"`
	Docker      Docker `yaml:"docker"`
}

type Odoo struct {
	Version string `yaml:"version"`
	Edition string `yaml:"edition"`
}

// Paths are optional overrides. Runivra normally reads these locations from
// the compose file and odoo.conf; set one here only when it guesses wrong.
type Paths struct {
	Config     string `yaml:"config,omitempty"`
	Custom     string `yaml:"custom,omitempty"`
	Enterprise string `yaml:"enterprise,omitempty"`
}

type Docker struct {
	ComposeFile     string `yaml:"compose_file"`
	OdooService     string `yaml:"odoo_service"`
	DatabaseService string `yaml:"database_service"`
}

func configPath(root string) string {
	return filepath.Join(root, DirName, ConfigName)
}

func Save(root string, config Config) error {
	config.Schema = SchemaVersion
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath(root), data, 0o644)
}

func Load(root string) (Config, error) {
	var config Config
	data, err := os.ReadFile(configPath(root))
	if err != nil {
		return config, err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("%s is not valid: %w", configPath(root), err)
	}
	if config.Schema != SchemaVersion {
		return config, fmt.Errorf("%s uses schema %d, but this version of Runivra understands schema %d", configPath(root), config.Schema, SchemaVersion)
	}
	return config, nil
}

func Find(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(configPath(current)); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", ErrNotFound
		}
		current = parent
	}
}
