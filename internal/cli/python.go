package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/GloriousPurposeVariant/runivra/internal/python"
	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

const pythonUsage = `Find or install the Python an Odoo version needs.

Usage:
  runivra python --odoo VERSION [options]

Each Odoo version is paired with the Python its official Docker image uses.
When that Python is missing, this offers to install it.

Options:
  --odoo VERSION   the Odoo version, for example 20.0 (required)
  --install        install without asking when that Python is missing
  --venv DIR       also create an isolated environment in DIR
`

func runPython(args []string) int {
	flags := flag.NewFlagSet("python", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var odoo, venv string
	var install bool
	flags.StringVar(&odoo, "odoo", "", "")
	flags.BoolVar(&install, "install", false, "")
	flags.StringVar(&venv, "venv", "", "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Print(pythonUsage)
			return 0
		}
		fmt.Println(paint(red, "runivra python:"), err)
		return 2
	}
	if odoo == "" {
		fmt.Println(paint(red, "runivra python:"), "--odoo is required, for example --odoo 20.0")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	path, ok := ensurePython(ctx, odoo, install, true)
	if !ok {
		return 1
	}
	if venv == "" {
		return 0
	}
	inside, err := python.CreateEnvironment(ctx, path, venv)
	if err != nil {
		fmt.Println(paint(red, "runivra python:"), err)
		return 1
	}
	fmt.Println(paint(green, "Environment ready:"), paint(cyan, inside))
	return 0
}

// ensurePython makes sure the Python for an Odoo version is installed and
// returns its path. It returns false when it is not available.
func ensurePython(ctx context.Context, odoo string, install bool, ask bool) (string, bool) {
	version, err := python.VersionFor(odoo)
	if err != nil {
		fmt.Println(paint(red, "runivra python:"), err, paint(dim, "(supported: "+fmt.Sprint(setup.SupportedVersions)+")"))
		return "", false
	}

	if path := python.Find(ctx, version); path != "" {
		fmt.Println(paint(green, "Python "+version+" for Odoo "+odoo+":"), paint(cyan, path))
		return path, true
	}

	fmt.Println(paint(yellow, "Odoo "+odoo+" needs Python "+version+", which is not installed on this computer."))
	if !install && ask && fancy {
		install = askYes("Install Python " + version + " now?")
	}
	if !install {
		fmt.Println(paint(dim, "Nothing was installed. Run \"runivra python --odoo "+odoo+" --install\" when you are ready."))
		return "", false
	}

	fmt.Println(paint(dim, "Installing Python "+version+". You may be asked for your password or for permission."))
	fmt.Println()
	if err := python.Install(ctx, os.Stdin, os.Stdout, version); err != nil {
		fmt.Println(paint(red, "Python was not installed:"), err)
		return "", false
	}
	path := python.Find(ctx, version)
	if path == "" {
		fmt.Println(paint(yellow, "Python "+version+" was installed but is not found yet. Open a new terminal and run this again."))
		return "", false
	}
	fmt.Println()
	fmt.Println(paint(green, "Python "+version+" is installed:"), paint(cyan, path))
	return path, true
}
