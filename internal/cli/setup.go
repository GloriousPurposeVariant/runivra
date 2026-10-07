package cli

import (
	"flag"
	"fmt"

	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

func runSetup(args []string) int {
	var req setup.Request

	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	flags.StringVar(&req.Environment, "env", "", "dev, staging or production")
	flags.StringVar(&req.Version, "version", "", "Odoo version, for example 19.0")
	flags.StringVar(&req.Path, "path", ".", "project folder, created if it does not exist")
	flags.StringVar(&req.Name, "name", "", "create a folder with this name inside the path and build there")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	problems := req.Problems()
	if len(problems) > 0 {
		fmt.Println("runivra setup: problems found:")
		for _, problem := range problems {
			fmt.Println("  ", problem)
		}
		return 2
	}

	if err := setup.CheckFolder(req.Folder()); err != nil {
		fmt.Println("runivra setup:", err)
		return 2
	}

	fmt.Println("Setup plan for", req.Environment, "with Odoo", req.Version)
	fmt.Println("Project folder:", req.Folder())
	for index, step := range setup.BuildPlan(req) {
		fmt.Printf("  %d. %-16s %s\n", index+1, step.Action, step.Target)
	}
	fmt.Println("Nothing was changed yet.")

	return 0
}
