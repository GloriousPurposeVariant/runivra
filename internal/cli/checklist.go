package cli

import (
	"fmt"

	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

var (
	boxTodo  = paint(dim, "[ ]")
	boxDone  = paint(green, "[✓]")
	boxFail  = paint(red, "[x]")
	boxLater = paint(yellow, "[-]")
)

func checklistLine(box string, step setup.Step) string {
	action := fmt.Sprintf("%-16s", step.Action)
	return "  " + box + " " + action + " " + paint(cyan, step.Target)
}

func printChecklist(steps []setup.Step) {
	for _, step := range steps {
		fmt.Println(checklistLine(boxTodo, step))
	}
}

func markStep(total int, index int, box string, step setup.Step) {
	up := total - index
	fmt.Printf("\x1b[%dA\r%s\x1b[%dB\r", up, checklistLine(box, step), up)
}
