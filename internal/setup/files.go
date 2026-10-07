package setup

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates
var templates embed.FS

func render(name string, req Request) ([]byte, error) {
	tmpl, err := template.ParseFS(templates, "templates/"+name)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, req); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func writeFile(step Step, req Request) error {
	content, err := render(step.Template, req)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(step.Target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(step.Target, content, 0o644)
}
