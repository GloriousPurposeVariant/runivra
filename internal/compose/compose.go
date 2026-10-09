package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type document struct {
	Services map[string]struct {
		Ports []struct {
			Target    int    `json:"target"`
			Published string `json:"published"`
		} `json:"ports"`
	} `json:"services"`
}

func parsePublishedPort(data []byte, service string, target int) (int, error) {
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return 0, err
	}
	entry, found := doc.Services[service]
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

// PublishedPort asks Docker Compose which port on this computer leads to the
// given port inside a service. Compose resolves the file exactly as it would
// when starting the stack, so the answer is what will really be used.
func PublishedPort(ctx context.Context, file string, service string, target int) (int, error) {
	output, err := exec.CommandContext(ctx, "docker", "compose", "-f", file, "config", "--format", "json").Output()
	if err != nil {
		return 0, fmt.Errorf("docker compose could not read %s: %w", file, err)
	}
	return parsePublishedPort(output, service, target)
}
