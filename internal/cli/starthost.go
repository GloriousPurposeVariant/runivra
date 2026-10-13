package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/GloriousPurposeVariant/runivra/internal/python"
	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

// hostCommand is how Odoo is started when it runs on this computer.
func hostCommand(req setup.Request) []string {
	folder := req.Folder()
	return []string{
		python.EnvironmentPython(runtime.GOOS, setup.EnvironmentFolder(req)),
		filepath.Join(folder, "odoo-bin"),
		"--config", filepath.Join(folder, "odoo.conf"),
	}
}

// startHostOdoo runs Odoo in this terminal until the user stops it.
func startHostOdoo(ctx context.Context, req setup.Request, options runOptions) int {
	command := hostCommand(req)
	start := options.start
	if !start && options.ask && fancy {
		start = askYesNo("Start Odoo now?")
	}
	if !start {
		fmt.Println(paint(dim, "To start it later, run:"))
		fmt.Println("  " + command[0] + " " + command[1] + " " + command[2] + " " + command[3])
		return 0
	}

	fmt.Println()
	fmt.Println(paint(bold, "Starting Odoo"), paint(dim, "at"), paint(cyan, fmt.Sprintf("http://localhost:%d", req.Port)))
	fmt.Println(paint(dim, "It runs in this terminal. Press Ctrl+C to stop it. The master password is \"dev-admin\"."))
	fmt.Println()

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = req.Folder()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		fmt.Println()
		fmt.Println(paint(dim, "Odoo was stopped."))
		return 0
	}
	if err != nil {
		fmt.Println(paint(red, "runivra setup:"), "Odoo stopped with an error:", err)
		return 1
	}
	return 0
}
