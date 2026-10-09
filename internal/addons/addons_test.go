package addons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/compose"
)

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	name := filepath.Join(parts...)
	if err := os.MkdirAll(name, 0o755); err != nil {
		t.Fatal(err)
	}
	return name
}

func write(t *testing.T, content string, parts ...string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(parts...), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolve(t *testing.T) {
	root := t.TempDir()
	custom := mkdir(t, root, "custom")
	extra := mkdir(t, root, "clients", "acme")
	enterprise := mkdir(t, root, "enterprise")
	mkdir(t, enterprise, "web_enterprise")

	volumes := []compose.Volume{
		{Type: "bind", Source: custom, Target: "/opt/odoo/custom"},
		{Type: "bind", Source: filepath.Join(root, "clients"), Target: "/mnt/clients"},
		{Type: "bind", Source: enterprise, Target: "/opt/odoo/enterprise/"},
		{Type: "bind", Source: filepath.Join(root, "odoo.conf"), Target: "/etc/odoo/odoo.conf"},
		{Type: "volume", Source: "odoo-data", Target: "/var/lib/odoo"},
	}
	folders := Resolve(" /opt/odoo/custom, /mnt/clients/acme ,/opt/odoo/enterprise,/usr/lib/python3/dist-packages/odoo/addons,", volumes)

	if len(folders) != 4 {
		t.Fatalf("got %d folders, want 4: %+v", len(folders), folders)
	}
	if folders[0].Host != custom || folders[0].Kind != KindCustom {
		t.Fatalf("first = %+v, want the custom folder", folders[0])
	}
	if folders[1].Host != extra || folders[1].Kind != KindCustom {
		t.Fatalf("second = %+v, want a folder below a mount to be found", folders[1])
	}
	if folders[2].Host != enterprise || folders[2].Kind != KindEnterprise {
		t.Fatalf("third = %+v, want Enterprise", folders[2])
	}
	if folders[3].Host != "" || folders[3].Kind != KindImage {
		t.Fatalf("fourth = %+v, want a folder that only exists in the image", folders[3])
	}

	if got := Key(root, folders); got != "clients/acme,custom" {
		t.Fatalf("Key() = %q, want the custom folders only, sorted", got)
	}
}

func TestRequirementProblems(t *testing.T) {
	root := t.TempDir()
	custom := mkdir(t, root, "custom")
	other := mkdir(t, root, "custom_proj")
	bare := mkdir(t, root, "no_requirements")
	write(t, "requests\n", custom, "requirements.txt")
	write(t, "pandas\n", other, "requirements.txt")
	write(t, "FROM odoo:20.0\nRUN pip3 install -r /tmp/project/custom/requirements.txt\n", root, "Dockerfile")
	write(t, "*\n!custom/requirements.txt\n", root, ".dockerignore")

	service := compose.Service{Build: &compose.Build{Context: root, Dockerfile: "Dockerfile"}}
	folders := []Folder{
		{Host: custom, Kind: KindCustom},
		{Host: other, Kind: KindCustom},
		{Host: bare, Kind: KindCustom},
	}

	problems := RequirementProblems(service, folders)
	if len(problems) != 1 || !strings.HasPrefix(problems[0], "custom_proj/requirements.txt is not installed") {
		t.Fatalf("problems = %q, want only custom_proj reported", problems)
	}

	write(t, "FROM odoo:20.0\nRUN pip3 install -r custom/requirements.txt -r custom_proj/requirements.txt\n", root, "Dockerfile")
	problems = RequirementProblems(service, folders)
	if len(problems) != 1 || !strings.Contains(problems[0], ".dockerignore") {
		t.Fatalf("problems = %q, want the .dockerignore exclusion reported", problems)
	}

	if got := RequirementProblems(compose.Service{Image: "odoo:20.0"}, folders); got != nil {
		t.Fatalf("a ready-made image has no Dockerfile to check, got %q", got)
	}
}
