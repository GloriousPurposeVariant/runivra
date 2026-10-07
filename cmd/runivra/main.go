package main

import (
	"github.com/GloriousPurposeVariant/runivra/internal/cli"
	"os"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
