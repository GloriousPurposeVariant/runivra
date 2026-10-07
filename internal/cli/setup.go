package cli

import (
	"errors"
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

	steps := setup.BuildPlan(req)
	fmt.Println(paint(bold, "Runivra setup"), paint(dim, "·"), req.Environment, paint(dim, "·"), "Odoo", req.Version)
	fmt.Println(paint(dim, "Project folder:"), paint(cyan, req.Folder()))
	fmt.Println()
	printChecklist(steps)

	for index, step := range steps {
		err := setup.Apply(step, req)
		if errors.Is(err, setup.ErrNotBuiltYet) {
			markStep(len(steps), index, boxLater, step)
			continue
		}
		if err != nil {
			markStep(len(steps), index, boxFail, step)
			fmt.Println(paint(red, "runivra setup:"), err)
			return 1
		}
		markStep(len(steps), index, boxDone, step)
	}

	fmt.Println()
	fmt.Println(paint(green, "Done."), paint(dim, "Steps marked [-] arrive in later milestones."))
	return 0

}
