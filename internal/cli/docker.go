package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

func askYesNo(question string) bool {
	fmt.Print(question + " [y/N] ")
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

func ensureDocker(ctx context.Context, install bool) {
	if setup.DetectDocker(ctx).Ready() {
		return
	}

	fmt.Println(paint(yellow, "Docker and Docker Compose are needed to run this project and were not found."))
	if !install && fancy {
		install = askYesNo("Install Docker now?")
	}
	if !install {
		fmt.Println(paint(dim, "Continuing without Docker. You can install it later from https://docs.docker.com/get-docker/"))
		fmt.Println()
		return
	}

	fmt.Println(paint(dim, "Installing Docker. You may be asked for your password or for permission."))
	if err := setup.InstallDocker(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Println(paint(red, "Docker was not installed:"), err)
		fmt.Println()
		return
	}

	if setup.DetectDocker(ctx).Ready() {
		fmt.Println(paint(green, "Docker is installed."))
	} else {
		fmt.Println(paint(yellow, "Docker was installed but is not ready yet. Start Docker Desktop or open a new terminal before running the project."))
	}
	fmt.Println()
}
