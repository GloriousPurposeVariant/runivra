package setup

import (
	"context"
	"fmt"
	"runtime"

	"github.com/GloriousPurposeVariant/runivra/internal/python"
)

// Warning is returned by a step that finished its work but left something
// for the user to look at. Setup carries on after a Warning.
type Warning struct {
	Summary string
	Details []string
}

func (w *Warning) Error() string {
	return w.Summary
}

// installPackages installs Odoo's Python requirements into the project's
// environment. A package that fails to build is reported and skipped, because
// on some systems a few of them need extra tools that only the user can add.
func installPackages(ctx context.Context, step Step, req Request, report func(Progress)) error {
	environment := python.EnvironmentPython(runtime.GOOS, EnvironmentFolder(req))
	progress := func(done int, total int, name string) {
		if report != nil && total > 0 {
			report(Progress{Label: name, Percent: done * 100 / total})
		}
	}

	failures, err := python.InstallRequirements(ctx, environment, step.Target, progress)
	if err != nil {
		return err
	}
	if len(failures) == 0 {
		return nil
	}

	warning := &Warning{Summary: fmt.Sprintf("%d Python package(s) could not be installed. Odoo may still start; features that need them will not work until they are installed.", len(failures))}
	for _, failure := range failures {
		warning.Details = append(warning.Details, failure.Requirement, "  "+failure.Reason)
	}
	warning.Details = append(warning.Details, "To retry one by hand: "+environment+" -m pip install <package>")
	return warning
}
