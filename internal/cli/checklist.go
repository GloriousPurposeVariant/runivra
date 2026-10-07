package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var (
	markTodo  = paint(dim, "○")
	markDone  = paint(green, "✔")
	markFail  = paint(red, "✘")
	markLater = paint(yellow, "–")
)

func checklistLine(mark string, step setup.Step, note string) string {
	action := fmt.Sprintf("%-16s", step.Action)
	return " " + mark + " " + action + " " + paint(cyan, step.Target) + "  " + paint(dim, note)
}

func printChecklist(steps []setup.Step) {
	for _, step := range steps {
		fmt.Println(checklistLine(markTodo, step, ""))
	}
}

func markStep(total int, index int, line string) {
	up := total - index
	fmt.Printf("\x1b[%dA\r%s\x1b[K\x1b[%dB\r", up, line, up)
}

func seconds(start time.Time) string {
	return fmt.Sprintf("%.1fs", time.Since(start).Seconds())
}

func resultMark(err error) string {
	switch {
	case err == nil:
		return markDone
	case errors.Is(err, setup.ErrNotBuiltYet):
		return markLater
	default:
		return markFail
	}
}

func runStep(total int, index int, step setup.Step, work func() error) error {
	start := time.Now()

	if !fancy {
		err := work()
		fmt.Println(checklistLine(resultMark(err), step, seconds(start)))
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- work()
	}()

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	frame := 0
	for {
		select {
		case err := <-done:
			markStep(total, index, checklistLine(resultMark(err), step, seconds(start)))
			return err
		case <-ticker.C:
			spinner := paint(cyan, spinnerFrames[frame%len(spinnerFrames)])
			markStep(total, index, checklistLine(spinner, step, seconds(start)))
			frame++
		}
	}
}
