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

	missing := req.Missing()
	if len(missing) > 0 {
		fmt.Println("runivra setup: missing options:")
		for _, option := range missing {
			fmt.Println("  ", option)
		}
		return 2
	}

	fmt.Println("Environment:", req.Environment)
	fmt.Println("Odoo Version:", req.Version)

	return 0
}
