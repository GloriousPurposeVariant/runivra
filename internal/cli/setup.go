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

	fmt.Println("Environment:", req.Environment)
	fmt.Println("Odoo Version:", req.Version)

	return 0
}
