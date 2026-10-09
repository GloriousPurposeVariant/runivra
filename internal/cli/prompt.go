package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// One reader is shared by every prompt. Creating a new one per question can
// swallow input meant for the next question.
var stdin = bufio.NewReader(os.Stdin)

func readLine() string {
	line, _ := stdin.ReadString('\n')
	return strings.TrimSpace(line)
}

// askYesNo treats an empty answer as no.
func askYesNo(question string) bool {
	fmt.Print(question + " [y/N] ")
	answer := strings.ToLower(readLine())
	return answer == "y" || answer == "yes"
}

// askYes treats an empty answer as yes.
func askYes(question string) bool {
	fmt.Print(question + " [Y/n] ")
	answer := strings.ToLower(readLine())
	return answer == "" || answer == "y" || answer == "yes"
}

// askText shows the current value and keeps it when the answer is empty.
func askText(label string, current string) string {
	shown := current
	if shown == "" {
		shown = "none"
	}
	fmt.Printf("  %s [%s]: ", label, paint(cyan, shown))
	if answer := readLine(); answer != "" {
		return answer
	}
	return current
}
