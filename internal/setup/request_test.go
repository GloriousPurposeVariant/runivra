package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFolderUsesNameWhenGiven(t *testing.T) {
	base := filepath.Join("projects", "clients")
	if got := (Request{Path: base}).Folder(); got != base {
		t.Fatalf("Folder() = %q, want %q", got, base)
	}
	want := filepath.Join(base, "shop")
	if got := (Request{Path: base, Name: "shop"}).Folder(); got != want {
		t.Fatalf("Folder() = %q, want %q", got, want)
	}
}

func TestProblemsRejectsPathAsName(t *testing.T) {
	req := Request{Environment: "dev", Version: "19.0", Name: "clients/shop"}
	if len(req.Problems()) != 1 {
		t.Fatalf("got %v, want one problem about --name", req.Problems())
	}
}

func TestCheckFolder(t *testing.T) {
	dir := t.TempDir()
	if err := CheckFolder(dir); err != nil {
		t.Fatalf("empty folder was rejected: %v", err)
	}
	if err := CheckFolder(filepath.Join(dir, "not-created-yet")); err != nil {
		t.Fatalf("a folder that does not exist yet was rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckFolder(dir); err == nil {
		t.Fatal("a folder with a file in it was accepted")
	}
}
