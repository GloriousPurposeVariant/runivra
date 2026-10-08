package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
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

var checklistRoot string

func shortTarget(target string) string {
	if checklistRoot == "" {
		return target
	}
	relative, err := filepath.Rel(checklistRoot, target)
	if err != nil || strings.HasPrefix(relative, "..") {
		return target
	}
	return relative
}

func checklistLine(mark string, step setup.Step, note string) string {
	action := fmt.Sprintf("%-16s", step.Action)
	return " " + mark + " " + action + " " + paint(cyan, shortTarget(step.Target)) + "  " + paint(dim, note)
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

func progressBar(p setup.Progress) string {
	const width = 20
	filled := p.Percent * width / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("%s %s %3d%%", p.Label, bar, p.Percent)
}

func runStep(total int, index int, step setup.Step, work func(report func(setup.Progress)) error) error {
	start := time.Now()

	if !fancy {
		err := work(func(setup.Progress) {})
		fmt.Println(checklistLine(resultMark(err), step, seconds(start)))
		return err
	}

	progress := make(chan setup.Progress, 16)
	report := func(p setup.Progress) {
		select {
		case progress <- p:
		default:
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- work(report)
	}()

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	frame := 0
	note := ""
	for {
		select {
		case err := <-done:
			markStep(total, index, checklistLine(resultMark(err), step, seconds(start)))
			return err
		case p := <-progress:
			note = progressBar(p) + "  "
		case <-ticker.C:
			spinner := paint(cyan, spinnerFrames[frame%len(spinnerFrames)])
			markStep(total, index, checklistLine(spinner, step, note+seconds(start)))
			frame++
		}
	}
}
