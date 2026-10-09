// Package odooconf reads values from an odoo.conf file. Runivra never keeps
// its own copy of these settings; it reads the file each time.
package odooconf

import (
	"bufio"
	"os"
	"strings"
)

// Read returns the settings in the [options] section.
func Read(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	inOptions := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";"):
		case strings.HasPrefix(line, "["):
			inOptions = line == "[options]"
		case inOptions:
			key, value, found := strings.Cut(line, "=")
			if found {
				values[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		}
	}
	return values, scanner.Err()
}
