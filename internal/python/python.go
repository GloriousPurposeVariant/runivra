// Package python finds or installs the Python that an Odoo version needs,
// and creates the isolated environment Odoo's packages are installed into.
package python

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// versions maps an Odoo version to the Python that Odoo's own Docker image
// ships with. That pairing is the one Odoo itself tests and releases.
var versions = map[string]string{
	"16.0": "3.10",
	"17.0": "3.10",
	"18.0": "3.12",
	"19.0": "3.12",
	"20.0": "3.12",
}

// VersionFor returns the Python version to use for an Odoo version.
func VersionFor(odoo string) (string, error) {
	version, found := versions[odoo]
	if !found {
		return "", fmt.Errorf("no Python version is known for Odoo %s", odoo)
	}
	return version, nil
}

// findCommand is the command that prints the path of a given Python version.
func findCommand(system string, version string) []string {
	if system == "windows" {
		return []string{"py", "-" + version, "-c", "import sys; print(sys.executable)"}
	}
	return []string{"python" + version, "-c", "import sys; print(sys.executable)"}
}

// Find returns the full path of the requested Python, or "" when it is not
// installed.
func Find(ctx context.Context, version string) string {
	command := findCommand(runtime.GOOS, version)
	output, err := exec.CommandContext(ctx, command[0], command[1:]...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// installCommands returns the commands that install a Python version,
// including what is needed to create environments and build packages.
func installCommands(system string, root bool, has func(name string) bool, version string) ([][]string, error) {
	sudo := func(args ...string) []string {
		if root {
			return args
		}
		return append([]string{"sudo"}, args...)
	}
	switch {
	case system == "windows" && has("winget"):
		return [][]string{{
			"winget", "install", "--exact", "--id", "Python.Python." + version,
			"--accept-source-agreements", "--accept-package-agreements",
		}}, nil
	case system == "darwin" && has("brew"):
		return [][]string{{"brew", "install", "python@" + version}}, nil
	case system == "linux" && has("apt-get"):
		return [][]string{
			sudo("apt-get", "update"),
			sudo("apt-get", "install", "-y", "python"+version, "python"+version+"-venv", "python"+version+"-dev"),
		}, nil
	case system == "linux" && has("dnf"):
		return [][]string{
			sudo("dnf", "install", "-y", "python"+version, "python"+version+"-devel"),
		}, nil
	}
	return nil, fmt.Errorf("automatic install is not supported on this system; install Python %s from https://www.python.org/downloads/ and run this again", version)
}

// Install runs the installer, passing its questions and progress through.
func Install(ctx context.Context, in io.Reader, out io.Writer, version string) error {
	has := func(name string) bool {
		_, err := exec.LookPath(name)
		return err == nil
	}
	commands, err := installCommands(runtime.GOOS, os.Geteuid() == 0, has, version)
	if err != nil {
		return err
	}
	for _, command := range commands {
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		cmd.Stdin = in
		cmd.Stdout = out
		cmd.Stderr = out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s failed: %w", command[0], err)
		}
	}
	return nil
}

// EnvironmentPython returns where the Python program lives inside an
// environment folder. Windows lays environments out differently.
func EnvironmentPython(system string, folder string) string {
	if system == "windows" {
		return filepath.Join(folder, "Scripts", "python.exe")
	}
	return filepath.Join(folder, "bin", "python")
}

// CreateEnvironment makes an isolated environment in folder using the given
// Python, and returns the path of the Python inside it. An environment that
// is already there is kept.
func CreateEnvironment(ctx context.Context, pythonPath string, folder string) (string, error) {
	inside := EnvironmentPython(runtime.GOOS, folder)
	if _, err := os.Stat(inside); err == nil {
		return inside, nil
	}
	output, err := exec.CommandContext(ctx, pythonPath, "-m", "venv", folder).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("the environment could not be created: %s", strings.TrimSpace(string(output)))
	}
	return inside, nil
}
