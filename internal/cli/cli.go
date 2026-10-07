package cli

import "fmt"

func Run(args []string) int {
	if len(args) == 0 {
		fmt.Println("Runivra: no command given. Try: runivra setup")
		return 0
	}

	command := args[0]
	if command == "setup" {
		return runSetup(args[1:])
	}
	fmt.Println("Unknown command:", command)
	return 2
}
