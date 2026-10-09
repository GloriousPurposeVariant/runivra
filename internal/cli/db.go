package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/GloriousPurposeVariant/runivra/internal/database"
	"github.com/GloriousPurposeVariant/runivra/internal/project"
	"github.com/GloriousPurposeVariant/runivra/internal/workspace"
)

const dbUsage = `Work with the databases of a Runivra project.

Usage:
  runivra db              list the databases and choose one
  runivra db list         list the databases, one per line
  runivra db use NAME     remember NAME as the database to work on

The chosen database is remembered for this project on this computer, so later
commands do not ask again.
`

// openProject finds and loads the project around the current folder. It
// prints the reason and returns false when there is none.
func openProject(command string) (string, project.Config, bool) {
	root, err := project.Find(".")
	if errors.Is(err, project.ErrNotFound) {
		fmt.Println("No Runivra project was found in this folder or any folder above it.")
		fmt.Println(paint(dim, "Run \"runivra init\" to register a project that already exists, or \"runivra setup\" to create a new one."))
		return "", project.Config{}, false
	}
	if err != nil {
		fmt.Println(paint(red, command+":"), err)
		return "", project.Config{}, false
	}
	config, err := project.Load(root)
	if err != nil {
		fmt.Println(paint(red, command+":"), err)
		return "", project.Config{}, false
	}
	return root, config, true
}

// listDatabases prints a helpful message and returns false when the list
// cannot be read.
func listDatabases(ctx context.Context, root string, config project.Config) ([]string, bool) {
	names, err := database.List(ctx, root, config)
	if errors.Is(err, database.ErrNotRunning) {
		fmt.Println(paint(yellow, "The database is not running."))
		fmt.Println(paint(dim, "Start the project with \"docker compose up -d\" in"), paint(cyan, root))
		return nil, false
	}
	if err != nil {
		fmt.Println(paint(red, "runivra db:"), err)
		return nil, false
	}
	return names, true
}

func runDB(args []string) int {
	action := ""
	if len(args) > 0 {
		action = args[0]
	}
	if action == "help" || action == "--help" || action == "-h" {
		fmt.Print(dbUsage)
		return 0
	}
	if action != "" && action != "list" && action != "use" {
		fmt.Printf("runivra db: unknown action %q\n\n%s", action, dbUsage)
		return 2
	}
	if action == "use" && len(args) < 2 {
		fmt.Println(paint(red, "runivra db use:"), "give the name of the database")
		return 2
	}

	root, config, ok := openProject("runivra db")
	if !ok {
		return 1
	}
	ctx := context.Background()
	names, ok := listDatabases(ctx, root, config)
	if !ok {
		return 1
	}
	state, err := project.LoadState(root)
	if err != nil {
		fmt.Println(paint(red, "runivra db:"), err)
		return 1
	}
	key := ""
	if view, err := workspace.Load(ctx, root, config); err == nil {
		key = view.AddonsKey()
		if action == "" {
			printProblems(view)
		}
	}

	switch action {
	case "list":
		for _, name := range names {
			fmt.Println(name)
		}
		return 0
	case "use":
		name, err := database.Choose(names, args[1], "")
		if err != nil {
			fmt.Println(paint(red, "runivra db use:"), err)
			return 1
		}
		return rememberDatabase(root, state, key, name)
	}

	if len(names) == 0 {
		fmt.Println("This project has no databases yet.")
		fmt.Println(paint(dim, "Create one from Odoo's database screen in the browser."))
		return 0
	}
	fmt.Println(paint(bold, config.Name), paint(dim, "· databases"))
	fmt.Println()
	for index, name := range names {
		mark := " "
		if name == state.Database(key) {
			mark = paint(green, "✔")
		}
		fmt.Printf("  %s %s %s\n", mark, paint(dim, fmt.Sprintf("%2d.", index+1)), name)
	}
	fmt.Println()
	if !fancy {
		return 0
	}

	current := state.Database(key)
	if current == "" {
		current = "none"
	}
	fmt.Printf("Choose a database by number or name [%s]: ", paint(cyan, current))
	answer := readLine()
	if answer == "" {
		return 0
	}
	if number, err := strconv.Atoi(answer); err == nil && number >= 1 && number <= len(names) {
		answer = names[number-1]
	}
	name, err := database.Choose(names, answer, "")
	if err != nil {
		fmt.Println(paint(red, "runivra db:"), err)
		return 1
	}
	return rememberDatabase(root, state, key, name)
}

func rememberDatabase(root string, state project.State, key string, name string) int {
	state.SetDatabase(key, name)

	if err := project.SaveState(root, state); err != nil {
		fmt.Println(paint(red, "runivra db:"), err)
		return 1
	}
	fmt.Println(paint(green, "Using database"), paint(cyan, name))
	return 0
}
