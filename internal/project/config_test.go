package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func sample() Config {
	return Config{
		Name:        "shop",
		Environment: "development",
		Odoo:        Odoo{Version: "20.0", Edition: "enterprise"},
		Paths:       Paths{Config: "odoo.conf", Custom: "custom", Enterprise: "enterprise"},
		Docker:      Docker{ComposeFile: "docker-compose.yml", OdooService: "web", DatabaseService: "db"},
	}
}

func TestSaveThenLoad(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, sample()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := sample()
	want.Schema = SchemaVersion
	if loaded != want {
		t.Fatalf("loaded = %+v, want %+v", loaded, want)
	}
}

func TestFindWalksUpFromASubfolder(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, sample()); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "custom", "my_module", "models")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := Find(deep)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if !sameFolder(t, found, root) {
		t.Fatalf("Find() = %q, want %q", found, root)
	}
}

func TestFindReportsWhenThereIsNoProject(t *testing.T) {
	_, err := Find(t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Find() error = %v, want ErrNotFound", err)
	}
}

func TestLoadRejectsAnUnknownSchema(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath(root), []byte("schema: 99\nname: shop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("a config written by a newer Runivra must not be read as if it were understood")
	}
}

// sameFolder compares two paths after resolving links, because the system's
// temporary folder is itself a link on some systems.
func sameFolder(t *testing.T, a string, b string) bool {
	t.Helper()
	resolvedA, err := filepath.EvalSymlinks(a)
	if err != nil {
		t.Fatal(err)
	}
	resolvedB, err := filepath.EvalSymlinks(b)
	if err != nil {
		t.Fatal(err)
	}
	return resolvedA == resolvedB
}
