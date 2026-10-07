package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("Runivra: no command given. Try: runivra setup")
		return
	}

	command := args[0]

	if command == "setup" {
		fmt.Println("Setup will start here.")
		return
	}

	fmt.Println("Unknown command:", command)
	os.Exit(2)
}
