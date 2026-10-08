package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

func startOdoo(ctx context.Context, req setup.Request, options runOptions) int {
	if !setup.DetectDocker(ctx).Ready() {
		fmt.Println(paint(dim, "Docker is not available, so Odoo was not started."))
		return 0
	}
	start := options.start
	if !start && options.ask && fancy {
		start = askYesNo("Start Odoo now?")
	}

	if !start {
		fmt.Println(paint(dim, "To start it later, run \"docker compose up -d --build\" in"), paint(cyan, req.Folder()))
		return 0
	}

	fmt.Println()
	fmt.Println(paint(bold, "Starting Odoo"))
	if err := setup.ComposeUp(ctx, req.Folder(), os.Stdout); err != nil {
		fmt.Println(paint(red, "runivra setup:"), "docker compose failed:", err)
		return 1
	}

	url := fmt.Sprintf("http://localhost:%d", req.Port)
	wait := setup.Step{Action: "wait for Odoo", Target: url}
	if fancy {
		fmt.Println()
		printChecklist([]setup.Step{wait})
	}
	err := runStep(1, 0, wait, func(func(setup.Progress)) error {
		return setup.WaitForOdoo(ctx, url, 4*time.Minute)
	})
	if err != nil {
		fmt.Println(paint(red, "runivra setup:"), err)
		return 1
	}

	fmt.Println()
	fmt.Println(paint(green, "Odoo is running at"), paint(cyan, url))
	fmt.Println(paint(dim, "Create your first database there. The master password is \"dev-admin\"."))
	return 0
}
