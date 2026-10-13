package python

import (
	"path/filepath"
	"strings"
	"testing"
)

func only(names ...string) func(string) bool {
	return func(name string) bool {
		for _, candidate := range names {
			if candidate == name {
				return true
			}
		}
		return false
	}
}

func TestVersionFor(t *testing.T) {
	for odoo, want := range map[string]string{"16.0": "3.10", "17.0": "3.10", "18.0": "3.12", "20.0": "3.12"} {
		got, err := VersionFor(odoo)
		if err != nil || got != want {
			t.Fatalf("VersionFor(%s) = %q, %v; want %s", odoo, got, err, want)
		}
	}
	if _, err := VersionFor("15.0"); err == nil {
		t.Fatal("an Odoo version Runivra does not know must return an error")
	}
}

func TestFindCommand(t *testing.T) {
	windows := findCommand("windows", "3.12")
	if windows[0] != "py" || windows[1] != "-3.12" {
		t.Fatalf("windows = %v; want the py launcher asked for 3.12", windows)
	}
	linux := findCommand("linux", "3.12")
	if linux[0] != "python3.12" {
		t.Fatalf("linux = %v; want python3.12", linux)
	}
}

func TestInstallCommands(t *testing.T) {
	windows, err := installCommands("windows", false, only("winget"), "3.12")
	if err != nil || !strings.Contains(strings.Join(windows[0], " "), "Python.Python.3.12") {
		t.Fatalf("windows = %v, %v", windows, err)
	}

	mac, err := installCommands("darwin", false, only("brew"), "3.12")
	if err != nil || mac[0][2] != "python@3.12" {
		t.Fatalf("darwin = %v, %v", mac, err)
	}

	debian, err := installCommands("linux", false, only("apt-get"), "3.12")
	joined := strings.Join(debian[1], " ")
	if err != nil || debian[1][0] != "sudo" || !strings.Contains(joined, "python3.12-venv") || !strings.Contains(joined, "python3.12-dev") {
		t.Fatalf("debian = %v, %v; want sudo, with the venv and dev packages", debian, err)
	}

	asRoot, err := installCommands("linux", true, only("apt-get"), "3.12")
	if err != nil || asRoot[1][0] != "apt-get" {
		t.Fatalf("debian as root = %v, %v; want no sudo", asRoot, err)
	}

	if _, err := installCommands("windows", false, only(), "3.12"); err == nil {
		t.Fatal("Windows without winget must return an error")
	}
}

func TestEnvironmentPython(t *testing.T) {
	if got := EnvironmentPython("windows", "venv"); got != filepath.Join("venv", "Scripts", "python.exe") {
		t.Fatalf("windows = %q", got)
	}
	if got := EnvironmentPython("linux", "venv"); got != filepath.Join("venv", "bin", "python") {
		t.Fatalf("linux = %q", got)
	}
}
