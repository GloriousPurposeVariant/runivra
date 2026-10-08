package setup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"time"
)

func PortIsFree(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func ComposeUp(ctx context.Context, folder string, out io.Writer) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "up", "--detach", "--build")
	cmd.Dir = folder
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

func WaitForOdoo(ctx context.Context, url string, limit time.Duration) error {
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err == nil {
			response.Body.Close()
			if response.StatusCode < 500 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return errors.New("cancelled")
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("odoo did not answer at %s within %s; run \"docker compose logs web\" in the project folder to see why", url, limit)
}
