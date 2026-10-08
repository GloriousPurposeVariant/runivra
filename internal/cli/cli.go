package cli

import "fmt"

const usage = `Runivra prepares and runs Odoo projects.

Usage:
  runivra <command>

Commands:
  setup    Create an Odoo project folder (opens a wizard when run without options)
  help     Show this help

Run "runivra setup --help" for the setup options.
`

func Run(args []string) int {
	if len(args) == 0 {
		fmt.Print(usage)
		return 0
	}

	switch args[0] {
	case "help", "--help", "-h":
		fmt.Print(usage)
		return 0
	case "setup":
		return runSetup(args[1:])
	}

	fmt.Printf("runivra: unknown command %q\n\n%s", args[0], usage)
	return 2
}
