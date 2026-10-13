package python

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Failure is one requirement that could not be installed.
type Failure struct {
	Requirement string
	Reason      string
}

var commentStart = regexp.MustCompile(`(^|\s)#.*$`)

// ParseRequirements returns the requirement lines of a requirements.txt,
// without comments, blank lines or pip options.
func ParseRequirements(text string) []string {
	var requirements []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(commentStart.ReplaceAllString(line, ""))
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		requirements = append(requirements, line)
	}
	return requirements
}

// packageName returns the name at the start of a requirement line.
func packageName(requirement string) string {
	end := strings.IndexAny(requirement, " =<>!~;[")
	if end < 0 {
		return requirement
	}
	return requirement[:end]
}

// lastLine picks the most useful line of pip's output: the last one that
// mentions an error, or failing that the last line of all.
func lastLine(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	chosen := strings.TrimSpace(lines[len(lines)-1])
	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if strings.HasPrefix(strings.ToLower(line), "error") {
			chosen = line
			break
		}
	}
	if len(chosen) > 200 {
		chosen = chosen[:200] + "..."
	}
	return chosen
}

// InstallRequirements installs each requirement on its own, so that one
// package that cannot be built does not stop the others. It returns the ones
// that failed. report is called before each package with how far along it is.
func InstallRequirements(ctx context.Context, environmentPython string, file string, report func(done int, total int, name string)) ([]Failure, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	requirements := ParseRequirements(string(content))

	pip := func(args ...string) (string, error) {
		full := append([]string{"-m", "pip", "install", "--disable-pip-version-check", "--no-input"}, args...)
		output, err := exec.CommandContext(ctx, environmentPython, full...).CombinedOutput()
		return string(output), err
	}

	// Building packages from source needs current versions of these.
	if output, err := pip("--upgrade", "pip", "setuptools", "wheel"); err != nil {
		return []Failure{{Requirement: "pip, setuptools, wheel", Reason: lastLine(output)}}, nil
	}

	var failures []Failure
	for index, requirement := range requirements {
		if ctx.Err() != nil {
			return failures, ctx.Err()
		}
		if report != nil {
			report(index, len(requirements), packageName(requirement))
		}
		if output, err := pip(requirement); err != nil {
			failures = append(failures, Failure{Requirement: requirement, Reason: lastLine(output)})
		}
	}
	return failures, nil
}
