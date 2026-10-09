package odooconf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "odoo.conf")
	content := "# generated\n[options]\ndb_user = odoo\ndb_password=s3cr=et\n; a comment\nworkers = 0\n\n[other]\ndb_user = ignored\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	values, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if values["db_user"] != "odoo" {
		t.Fatalf("db_user = %q, want odoo; other sections must be ignored", values["db_user"])
	}
	if values["db_password"] != "s3cr=et" {
		t.Fatalf("db_password = %q, want the value kept whole after the first equals sign", values["db_password"])
	}
	if values["workers"] != "0" || len(values) != 3 {
		t.Fatalf("values = %v", values)
	}
}

func TestReadMissingFile(t *testing.T) {
	if _, err := Read(filepath.Join(t.TempDir(), "nope.conf")); err == nil {
		t.Fatal("a missing file must return an error")
	}
}
