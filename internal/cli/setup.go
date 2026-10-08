package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/GloriousPurposeVariant/runivra/internal/setup"
	"github.com/GloriousPurposeVariant/runivra/internal/tui"
)

type runOptions struct {
	installDocker bool
	start         bool
	ask           bool
}

func runSetup(args []string) int {

	if len(args) == 0 {
		answers, ok, err := tui.Run(tui.Options{
			DockerMissing: !setup.DetectDocker(context.Background()).Ready(),
			PortIsFree:    setup.PortIsFree,
		})

		if err != nil {
			fmt.Println(paint(red, "runivra setup:"), err)
			return 1
		}
		if !ok {
			fmt.Println("Setup cancelled.")
			return 0
		}
		path := answers.Path
		if path == "" {
			path = "."
		}
		return runPlan(setup.Request{
			Environment:     answers.Environment,
			Version:         answers.Version,
			Path:            path,
			Name:            answers.Name,
			EnterpriseToken: answers.EnterpriseToken,
			EnterprisePath:  answers.EnterprisePath,
			CustomRepo:      answers.CustomRepo,
			CustomBranch:    answers.CustomBranch,
			CustomToken:     answers.CustomToken,
			Port:            answers.Port,
		}, runOptions{installDocker: answers.InstallDocker, start: answers.Start, ask: false})

	}

	var req setup.Request
	var installDocker bool
	var start bool

	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	flags.StringVar(&req.Environment, "env", "", "dev, staging or production")
	flags.StringVar(&req.Version, "version", "", "Odoo version, for example 19.0")
	flags.StringVar(&req.Path, "path", ".", "project folder, created if it does not exist")
	flags.StringVar(&req.Name, "name", "", "create a folder with this name inside the path and build there")
	flags.StringVar(&req.EnterpriseToken, "enterprise-token", os.Getenv("RUNIVRA_ENTERPRISE_TOKEN"), "Git token used to clone Odoo Enterprise")
	flags.StringVar(&req.EnterprisePath, "enterprise-path", "", "copy Odoo Enterprise from this local folder")
	flags.StringVar(&req.CustomRepo, "custom-repo", "", "Git repository of your custom addons")
	flags.StringVar(&req.CustomBranch, "custom-branch", "", "branch of the custom addons (default: the Odoo version)")
	flags.StringVar(&req.CustomToken, "custom-token", os.Getenv("RUNIVRA_CUSTOM_TOKEN"), "Git token for a private custom addons repository")
	flags.BoolVar(&installDocker, "install-docker", false, "install Docker without asking when it is missing")
	flags.IntVar(&req.Port, "port", 8069, "port on this computer where Odoo will answer")
	flags.BoolVar(&start, "start", false, "start Odoo after setup without asking")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	return runPlan(req, runOptions{installDocker: installDocker, start: start, ask: true})

}

func runPlan(req setup.Request, options runOptions) int {
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

	if err := setup.CheckSources(req); err != nil {
		fmt.Println(paint(red, "runivra setup:"), err)
		return 2
	}

	if req.IsDevelopment() && !setup.PortIsFree(req.Port) {
		fmt.Println(paint(red, "runivra setup:"), fmt.Sprintf("port %d is already in use; choose another with --port", req.Port))
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ensureDocker(ctx, options)

	steps := setup.BuildPlan(req)
	fmt.Println(paint(bold, "Runivra setup"), paint(dim, "·"), req.Environment, paint(dim, "·"), "Odoo", req.Version)
	fmt.Println(paint(dim, "Project folder:"), paint(cyan, req.Folder()))
	fmt.Println()
	if fancy {
		printChecklist(steps)
		fmt.Print(hideCursor)
		defer fmt.Print(showCursor)
	}

	later := 0
	for index, step := range steps {
		err := runStep(len(steps), index, step, func(report func(setup.Progress)) error {
			return setup.Apply(ctx, step, req, report)
		})

		if errors.Is(err, setup.ErrNotBuiltYet) {
			later++
			continue
		}
		if err != nil {
			fmt.Println(paint(red, "runivra setup:"), err)
			return 1
		}
	}

	fmt.Print(showCursor)
	fmt.Println()
	if later > 0 {
		fmt.Println(paint(green, "Done."), paint(dim, "Steps marked – arrive in later milestones."))
		return 0
	}
	fmt.Println(paint(green, "Done."))
	if req.IsDevelopment() {
		return startOdoo(ctx, req, options)
	}
	return 0

}
