package main

import (
	"os"
	"github.com/GloriousPurposeVariant/runivra/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
