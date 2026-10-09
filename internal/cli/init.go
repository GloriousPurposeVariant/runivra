package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/GloriousPurposeVariant/runivra/internal/adopt"
	"github.com/GloriousPurposeVariant/runivra/internal/compose"
	"github.com/GloriousPurposeVariant/runivra/internal/project"
	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

const initUsage = `Register an Odoo project with Runivra.

Usage:
  runivra init [options] [folder]

In a folder that already holds an Odoo project, init reads the compose file,
shows what it found and saves it to .runivra/config.yml. Nothing else in the
project is changed. In an empty folder it starts a new project instead.

Options:
  --yes      save what was detected without asking
  --force    detect again even when the project is already registered
`

func runInit(args []string) int {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var yes, force bool
	flags.BoolVar(&yes, "yes", false, "")
	flags.BoolVar(&force, "force", false, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Print(initUsage)
			return 0
		}
		fmt.Println(paint(red, "runivra init:"), err)
		fmt.Println("Run \"runivra init --help\" to see the options.")
		return 2
	}

	folder := "."
	if flags.NArg() > 0 {
		folder = flags.Arg(0)
	}
	root, err := filepath.Abs(folder)
	if err != nil {
		fmt.Println(paint(red, "runivra init:"), err)
		return 2
	}

	if _, err := project.Load(root); err == nil && !force {
		fmt.Println(paint(cyan, root), "is already a Runivra project.")
		fmt.Println(paint(dim, "Run \"runivra info\" to see it, or \"runivra init --force\" to detect it again."))
		return 0
	}

	composeName := adopt.FindComposeFile(root)
	if composeName == "" {
		if setup.CheckFolder(root) == nil {
			fmt.Println(paint(dim, "This folder is empty, so there is nothing to register. Starting a new project."))
			return runSetup(nil)
		}
		fmt.Println(paint(red, "runivra init:"), "no compose file was found in", root)
		fmt.Println(paint(dim, "init registers projects that run with Docker Compose. For a new project, run \"runivra setup\"."))
		return 1
	}

	file, err := compose.Load(context.Background(), filepath.Join(root, composeName))
	if err != nil {
		fmt.Println(paint(red, "runivra init:"), err)
		fmt.Println(paint(dim, "Docker must be installed and the compose file must be valid."))
		return 1
	}

	config := adopt.Detect(root, composeName, file)
	fmt.Println(paint(bold, "Runivra init"), paint(dim, "·"), paint(cyan, root))
	fmt.Println()
	printConfig(config)
	fmt.Println()

	if !yes && !askYes("Save this?") {
		fmt.Println()
		fmt.Println(paint(dim, "Press Enter to keep a value, or type a new one."))
		config = editConfig(config)
		fmt.Println()
		printConfig(config)
		fmt.Println()
		if !askYes("Save this?") {
			fmt.Println("Nothing was saved.")
			return 0
		}
	}

	if config.Docker.OdooService == "" {
		fmt.Println(paint(red, "runivra init:"), "the Odoo service is required; nothing was saved.")
		return 1
	}
	if err := project.Save(root, config); err != nil {
		fmt.Println(paint(red, "runivra init:"), err)
		return 1
	}
	fmt.Println(paint(green, "Saved"), paint(cyan, filepath.Join(root, project.DirName, project.ConfigName)))
	return 0
}

func shownValue(value string) string {
	if value == "" {
		return paint(yellow, "not found")
	}
	return value
}

func printConfig(config project.Config) {
	row := func(label string, value string) {
		fmt.Println("  " + paint(dim, fmt.Sprintf("%-18s", label)) + " " + shownValue(value))
	}
	row("Name", config.Name)
	row("Environment", config.Environment)
	row("Odoo version", config.Odoo.Version)
	row("Edition", config.Odoo.Edition)
	row("Compose file", config.Docker.ComposeFile)
	row("Odoo service", config.Docker.OdooService)
	row("Database service", config.Docker.DatabaseService)
	row("Odoo config", config.Paths.Config)
	row("Custom addons", config.Paths.Custom)
	row("Enterprise", config.Paths.Enterprise)
}

func editConfig(config project.Config) project.Config {
	config.Name = askText("Name", config.Name)
	config.Environment = askText("Environment", config.Environment)
	config.Odoo.Version = askText("Odoo version", config.Odoo.Version)
	config.Odoo.Edition = askText("Edition", config.Odoo.Edition)
	config.Docker.OdooService = askText("Odoo service", config.Docker.OdooService)
	config.Docker.DatabaseService = askText("Database service", config.Docker.DatabaseService)
	config.Paths.Config = askText("Odoo config", config.Paths.Config)
	config.Paths.Custom = askText("Custom addons", config.Paths.Custom)
	config.Paths.Enterprise = askText("Enterprise", config.Paths.Enterprise)
	return config
}
