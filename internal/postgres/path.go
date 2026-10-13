package postgres

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// onPath reports whether folder is one of the entries of a PATH value.
// Windows ignores capitals and a trailing backslash, so this does too.
func onPath(pathValue string, folder string) bool {
	clean := func(entry string) string {
		return strings.ToLower(strings.TrimRight(strings.TrimSpace(entry), `\/`))
	}
	for _, entry := range filepath.SplitList(pathValue) {
		if clean(entry) == clean(folder) {
			return true
		}
	}
	return false
}

// AddToPath makes "psql" work by name in new terminals. It reports whether it
// changed anything. Runivra itself does not need this; it finds psql by its
// full path.
func AddToPath(ctx context.Context, psql string) (bool, error) {
	if _, err := exec.LookPath("psql"); err == nil {
		return false, nil
	}
	folder := filepath.Dir(psql)

	switch runtime.GOOS {
	case "windows":
		read := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "[Environment]::GetEnvironmentVariable('Path', 'User')")
		output, err := read.Output()
		if err != nil {
			return false, err
		}
		current := strings.TrimSpace(string(output))
		if onPath(current, folder) {
			return false, nil
		}
		updated := folder
		if current != "" {
			updated = strings.TrimRight(current, ";") + ";" + folder
		}
		write := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "[Environment]::SetEnvironmentVariable('Path', $env:RUNIVRA_NEW_PATH, 'User')")
		write.Env = append(os.Environ(), "RUNIVRA_NEW_PATH="+updated)
		return true, write.Run()
	case "darwin":
		return true, exec.CommandContext(ctx, "brew", "link", "--force", "postgresql@"+Version).Run()
	}
	return false, nil
}
