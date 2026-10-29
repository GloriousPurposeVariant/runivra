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
or for Odoo installed from source without Docker, odoo-bin and odoo.conf. It
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

	var config project.Config
	composeName := adopt.FindComposeFile(root)
	switch {
	case composeName != "":
		file, err := compose.Load(context.Background(), filepath.Join(root, composeName))
		if err != nil {
			fmt.Println(paint(red, "runivra init:"), err)
			fmt.Println(paint(dim, "Docker must be installed and the compose file must be valid."))
			return 1
		}
		config = adopt.Detect(root, composeName, file)
	case adopt.IsOdooSource(root):
		config = adopt.DetectHost(root)
	case setup.CheckFolder(root) == nil:
		fmt.Println(paint(dim, "This folder is empty, so there is nothing to register. Starting a new project."))
		return runSetup(nil)
	default:
		fmt.Println(paint(red, "runivra init:"), "no compose file and no odoo-bin were found in", root)
		fmt.Println(paint(dim, "Run init in the folder that holds docker-compose.yml or the Odoo source. For a new project, run \"runivra setup\"."))
		return 1
	}
	host := config.Odoo.Runtime == "host"
	detected := config.Paths

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

	if !host && config.Docker.OdooService == "" {
		fmt.Println(paint(red, "runivra init:"), "the Odoo service is required; nothing was saved.")
		return 1
	}
	if host && config.Paths.Config == "" {
		fmt.Println(paint(red, "runivra init:"), "the Odoo config file is required; nothing was saved.")
		return 1
	}
	// Locations are read from the project's files each time. Only a value the
	// user corrected by hand is kept, as an override. Without a compose file
	// nothing else points to odoo.conf, so a host project always keeps it.
	if !host && config.Paths.Config == detected.Config {
		config.Paths.Config = ""
	}
	if config.Paths.Custom == detected.Custom {
		config.Paths.Custom = ""
	}
	if config.Paths.Enterprise == detected.Enterprise {
		config.Paths.Enterprise = ""
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
	if config.Odoo.Runtime == "host" {
		row("Runs on", "this computer, without Docker")
		row("Python environment", config.Paths.Python)
	} else {
		row("Compose file", config.Docker.ComposeFile)
		row("Odoo service", config.Docker.OdooService)
		row("Database service", config.Docker.DatabaseService)
	}
	row("Odoo config", config.Paths.Config)
	row("Custom addons", config.Paths.Custom)
	row("Enterprise", config.Paths.Enterprise)
}

func editConfig(config project.Config) project.Config {
	config.Name = askText("Name", config.Name)
	config.Environment = askText("Environment", config.Environment)
	config.Odoo.Version = askText("Odoo version", config.Odoo.Version)
	config.Odoo.Edition = askText("Edition", config.Odoo.Edition)
	if config.Odoo.Runtime == "host" {
		config.Paths.Python = askText("Python environment", config.Paths.Python)
	} else {
		config.Docker.OdooService = askText("Odoo service", config.Docker.OdooService)
		config.Docker.DatabaseService = askText("Database service", config.Docker.DatabaseService)
	}
	config.Paths.Config = askText("Odoo config", config.Paths.Config)
	config.Paths.Custom = askText("Custom addons", config.Paths.Custom)
	config.Paths.Enterprise = askText("Enterprise", config.Paths.Enterprise)
	return config
}
