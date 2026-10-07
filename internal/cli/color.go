package cli

import "os"

const (
	reset  = "\x1b[0m"
	bold   = "\x1b[1m"
	dim    = "\x1b[2m"
	red    = "\x1b[31m"
	green  = "\x1b[32m"
	yellow = "\x1b[33m"
	cyan   = "\x1b[36m"

	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
)

var fancy = isTerminal()

func isTerminal() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func paint(color string, text string) string {
	if !fancy {
		return text
	}
	return color + text + reset
}
