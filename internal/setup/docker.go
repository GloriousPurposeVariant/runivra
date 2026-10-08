package setup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

const dockerDocs = "https://docs.docker.com/get-docker/"

type DockerStatus struct {
	Engine  bool
	Compose bool
}

func (s DockerStatus) Ready() bool {
	return s.Engine && s.Compose
}

func DetectDocker(ctx context.Context) DockerStatus {
	var status DockerStatus
	status.Engine = exec.CommandContext(ctx, "docker", "--version").Run() == nil
	if status.Engine {
		status.Compose = exec.CommandContext(ctx, "docker", "compose", "version").Run() == nil
	}
	return status
}

func dockerInstallCommand(system string, root bool) ([]string, error) {
	switch system {
	case "windows":
		return []string{"winget", "install", "--exact", "--id", "Docker.DockerDesktop",
			"--accept-source-agreements", "--accept-package-agreements"}, nil
	case "darwin":
		return []string{"brew", "install", "--cask", "docker"}, nil
	case "linux":
		script := "curl -fsSL https://get.docker.com | sh"
		if root {
			return []string{"sh", "-c", script}, nil
		}
		return []string{"sudo", "sh", "-c", script}, nil
	default:
		return nil, fmt.Errorf("automatic install is not supported on %s; see %s", system, dockerDocs)
	}
}

func InstallDocker(ctx context.Context, in io.Reader, out io.Writer) error {
	command, err := dockerInstallCommand(runtime.GOOS, os.Geteuid() == 0)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath(command[0]); err != nil {
		return fmt.Errorf("%s was not found, so Docker cannot be installed automatically; see %s", command[0], dockerDocs)
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}
