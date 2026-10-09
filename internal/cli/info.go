package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/GloriousPurposeVariant/runivra/internal/compose"
	"github.com/GloriousPurposeVariant/runivra/internal/project"
)

func runInfo(args []string) int {
	start := "."
	if len(args) > 0 {
		start = args[0]
	}

	root, err := project.Find(start)
	if errors.Is(err, project.ErrNotFound) {
		fmt.Println("No Runivra project was found in", paint(cyan, start), "or any folder above it.")
		fmt.Println(paint(dim, "Run \"runivra setup\" to create one."))
		return 1
	}
	if err != nil {
		fmt.Println(paint(red, "runivra info:"), err)
		return 1
	}
	config, err := project.Load(root)
	if err != nil {
		fmt.Println(paint(red, "runivra info:"), err)
		return 1
	}

	inside := func(relative string) string {
		return filepath.Join(root, filepath.FromSlash(relative))
	}
	row := func(label string, value string) {
		fmt.Println("  " + paint(dim, fmt.Sprintf("%-14s", label)) + " " + value)
	}

	address := paint(dim, "unknown (Docker could not read the compose file)")
	port, err := compose.PublishedPort(context.Background(), inside(config.Docker.ComposeFile), config.Docker.OdooService, 8069)
	if err == nil {
		address = fmt.Sprintf("http://localhost:%d", port)
	}

	fmt.Println(paint(bold, config.Name), paint(dim, "·"), config.Environment, paint(dim, "·"), "Odoo", config.Odoo.Version, config.Odoo.Edition)
	fmt.Println()
	row("Folder", paint(cyan, root))
	row("Address", address)
	row("Odoo config", inside(config.Paths.Config))
	row("Custom addons", inside(config.Paths.Custom))
	row("Enterprise", inside(config.Paths.Enterprise))
	row("Compose file", inside(config.Docker.ComposeFile))
	row("Services", config.Docker.OdooService+" (Odoo), "+config.Docker.DatabaseService+" (database)")
	return 0
}
