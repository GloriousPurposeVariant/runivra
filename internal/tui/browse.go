package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	optUseFolder = "✔ Use this folder"
	optParent    = "↑ Parent folder"
	optMore      = "… Show more"
	folderMark   = "📁 "
	pageSize     = 7
)

type browser struct {
	dir     string
	folders []string
	filter  string
	limit   int
}

func listFolders(dir string) []string {
	if dir == "" {
		return listDrives()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			names = append(names, entry.Name())
		}
	}
	return names
}

func listDrives() []string {
	var drives []string
	for letter := 'A'; letter <= 'Z'; letter++ {
		drive := string(letter) + `:\`
		if _, err := os.Stat(drive); err == nil {
			drives = append(drives, drive)
		}
	}
	return drives
}

func (m *model) open(dir string) {
	m.browser = browser{dir: dir, folders: listFolders(dir), limit: pageSize}
	m.cursor = 0
}

func (m model) startFolder() string {
	if saved := m.answers[m.current().key]; saved != "" {
		return saved
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

func (m model) hasParent() bool {
	dir := m.browser.dir
	if dir == "" {
		return false
	}
	return filepath.Dir(dir) != dir || runtime.GOOS == "windows"
}

func (m model) matches() []string {
	filter := strings.ToLower(m.browser.filter)
	var result []string
	for _, name := range m.browser.folders {
		if strings.Contains(strings.ToLower(name), filter) {
			result = append(result, name)
		}
	}
	return result
}

func (m model) fixedOptions() []string {
	var list []string
	if m.browser.dir != "" {
		list = append(list, optUseFolder)
	}
	if m.hasParent() {
		list = append(list, optParent)
	}
	return list
}

func (m model) browseOptions() []string {
	list := m.fixedOptions()
	matches := m.matches()
	shown := matches
	if len(shown) > m.browser.limit {
		shown = shown[:m.browser.limit]
	}
	for _, name := range shown {
		list = append(list, folderMark+name)
	}
	if hidden := len(matches) - len(shown); hidden > 0 {
		list = append(list, fmt.Sprintf("%s (%d more)", optMore, hidden))
	}
	return list
}

func (m model) options() []string {
	q := m.current()
	if q.browse {
		return m.browseOptions()
	}
	return q.options
}

func (m *model) search(filter string) {
	m.browser.filter = filter
	m.browser.limit = pageSize
	m.cursor = 0
	if filter != "" && len(m.matches()) > 0 {
		m.cursor = len(m.fixedOptions())
	}
}

func (m *model) pick() bool {
	option := m.options()[m.cursor]
	switch {
	case option == optUseFolder:
		m.answers[m.current().key] = m.browser.dir
		return true
	case option == optParent:
		parent := filepath.Dir(m.browser.dir)
		if parent == m.browser.dir {
			parent = ""
		}
		m.open(parent)
	case strings.HasPrefix(option, optMore):
		m.browser.limit += pageSize
	default:
		m.open(filepath.Join(m.browser.dir, strings.TrimPrefix(option, folderMark)))
	}
	return false
}

func trimLast(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	return string(runes[:len(runes)-1])
}
