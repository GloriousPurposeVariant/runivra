package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// File is the part of a compose file Runivra cares about, as Docker Compose
// reports it after resolving variables and relative paths.
type File struct {
	Services map[string]Service `json:"services"`
}

type Service struct {
	Image   string   `json:"image"`
	Build   *Build   `json:"build"`
	Ports   []Port   `json:"ports"`
	Volumes []Volume `json:"volumes"`
}

type Build struct {
	Context    string `json:"context"`
	Dockerfile string `json:"dockerfile"`
}

type Port struct {
	Target    int    `json:"target"`
	Published string `json:"published"`
}

type Volume struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func Parse(data []byte) (File, error) {
	var file File
	err := json.Unmarshal(data, &file)
	return file, err
}

// Load asks Docker Compose to read a compose file. Compose resolves it exactly
// as it would when starting the stack, so the result is what will really run.
func Load(ctx context.Context, path string) (File, error) {
	output, err := exec.CommandContext(ctx, "docker", "compose", "-f", path, "config", "--format", "json").Output()
	if err != nil {
		return File{}, fmt.Errorf("docker compose could not read %s: %w", path, err)
	}
	return Parse(output)
}

// PublishedPort returns the port on this computer that leads to the given
// port inside a service.
func (f File) PublishedPort(service string, target int) (int, error) {
	entry, found := f.Services[service]
	if !found {
		return 0, fmt.Errorf("the compose file has no service named %q", service)
	}
	for _, port := range entry.Ports {
		if port.Target == target {
			return strconv.Atoi(port.Published)
		}
	}
	return 0, fmt.Errorf("service %q does not publish port %d", service, target)
}

func PublishedPort(ctx context.Context, path string, service string, target int) (int, error) {
	file, err := Load(ctx, path)
	if err != nil {
		return 0, err
	}
	return file.PublishedPort(service, target)
}
