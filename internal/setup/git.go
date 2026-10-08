package setup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Progress struct {
	Label   string
	Percent int
}

var progressPattern = regexp.MustCompile(`^(?:remote: )?([A-Za-z ]+):\s+(\d+)%`)

type progressWriter struct {
	report func(Progress)
	buffer string
	last   string
}

func (w *progressWriter) Write(data []byte) (int, error) {
	w.buffer += string(data)
	for {
		cut := strings.IndexAny(w.buffer, "\r\n")
		if cut < 0 {
			break
		}
		line := strings.TrimSpace(w.buffer[:cut])
		w.buffer = w.buffer[cut+1:]
		if line == "" {
			continue
		}
		match := progressPattern.FindStringSubmatch(line)
		if match == nil {
			w.last = line
			continue
		}
		percent, _ := strconv.Atoi(match[2])
		w.report(Progress{Label: match[1], Percent: percent})
	}
	return len(data), nil
}

func (w *progressWriter) lastMessage() string {
	if rest := strings.TrimSpace(w.buffer); rest != "" {
		return rest
	}
	return w.last
}

func clone(ctx context.Context, step Step, report func(Progress)) error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git is not installed or not on the PATH; install it from https://git-scm.com and run setup again")
	}
	if report == nil {
		report = func(Progress) {}
	}

	cmd := exec.CommandContext(ctx, "git",
		"-c", "core.longpaths=true",
		"clone", "--progress", "--depth", "1", "--single-branch",
		"--branch", step.Branch,
		step.URL, step.Target,
	)
	watcher := &progressWriter{report: report}
	cmd.Stderr = watcher
	err := cmd.Run()
	if err == nil {
		return nil
	}

	emptyFolder(step.Target)
	if ctx.Err() != nil {
		return errors.New("cancelled; the partial download was removed")
	}
	return fmt.Errorf("git clone failed: %s", watcher.lastMessage())
}

func emptyFolder(path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}
	for _, entry := range entries {
		os.RemoveAll(filepath.Join(path, entry.Name()))
	}
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
