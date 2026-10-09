package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/GloriousPurposeVariant/runivra/internal/addons"
	"github.com/GloriousPurposeVariant/runivra/internal/project"
	"github.com/GloriousPurposeVariant/runivra/internal/workspace"
)

// printProblems lists what Runivra found wrong with the project's files. It
// is called by every command that loads a project, so a problem is seen
// whichever command the user happens to run.
func printProblems(view workspace.View) {
	for _, problem := range view.Problems {
		fmt.Println(paint(yellow, "  ! "+problem))
	}
	if len(view.Problems) > 0 {
		fmt.Println()
	}
}

func runInfo(args []string) int {
	start := "."
	if len(args) > 0 {
		start = args[0]
	}

	root, err := project.Find(start)
	if errors.Is(err, project.ErrNotFound) {
		fmt.Println("No Runivra project was found in", paint(cyan, start), "or any folder above it.")
		fmt.Println(paint(dim, "Run \"runivra init\" to register a project that already exists, or \"runivra setup\" to create a new one."))
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

	row := func(label string, value string) {
		fmt.Println("  " + paint(dim, fmt.Sprintf("%-14s", label)) + " " + value)
	}
	fmt.Println(paint(bold, config.Name), paint(dim, "·"), config.Environment, paint(dim, "·"), "Odoo", config.Odoo.Version, config.Odoo.Edition)
	fmt.Println()
	row("Folder", paint(cyan, root))
	row("Compose file", config.Docker.ComposeFile)
	row("Services", config.Docker.OdooService+" (Odoo), "+config.Docker.DatabaseService+" (database)")

	view, err := workspace.Load(context.Background(), root, config)
	if err != nil {
		fmt.Println()
		fmt.Println(paint(yellow, "  ! The rest could not be read: "+err.Error()))
		return 1
	}

	address := paint(dim, "the Odoo service publishes no web port")
	if view.Port != 0 {
		address = fmt.Sprintf("http://localhost:%d", view.Port)
	}
	conf := paint(yellow, "not found")
	if view.ConfPath != "" {
		conf = addons.Relative(root, view.ConfPath)
	}
	row("Address", address)
	row("Odoo config", conf)

	state, _ := project.LoadState(root)
	database := state.Database(view.AddonsKey())
	if database == "" {
		database = paint(dim, "none chosen; run \"runivra db\"")
	}
	row("Database", database)

	fmt.Println()
	fmt.Println("  " + paint(dim, "Addons folders, in the order Odoo loads them"))
	if len(view.Addons) == 0 {
		fmt.Println("    " + paint(yellow, "none found in addons_path"))
	}
	for _, folder := range view.Addons {
		location := paint(dim, folder.Container+" (inside the image)")
		if folder.Host != "" {
			location = paint(cyan, addons.Relative(root, folder.Host))
		}
		fmt.Println("    " + fmt.Sprintf("%-11s", folder.Kind) + " " + location)
	}
	fmt.Println()
	printProblems(view)
	return 0
}
