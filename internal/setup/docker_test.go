package setup

import "testing"

func TestDockerInstallCommand(t *testing.T) {
	windows, err := dockerInstallCommand("windows", false)
	if err != nil || windows[0] != "winget" {
		t.Fatalf("windows = %v, %v; want winget", windows, err)
	}

	mac, err := dockerInstallCommand("darwin", false)
	if err != nil || mac[0] != "brew" {
		t.Fatalf("darwin = %v, %v; want brew", mac, err)
	}

	user, err := dockerInstallCommand("linux", false)
	if err != nil || user[0] != "sudo" {
		t.Fatalf("linux as a normal user = %v, %v; want sudo first", user, err)
	}

	root, err := dockerInstallCommand("linux", true)
	if err != nil || root[0] != "sh" {
		t.Fatalf("linux as root = %v, %v; want sh without sudo", root, err)
	}

	if _, err := dockerInstallCommand("plan9", false); err == nil {
		t.Fatal("an unsupported system must return an error")
	}
}
