package setup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/project"
	"gopkg.in/yaml.v3"
)

// composeShape is just enough of a compose file to check its structure.
type composeShape struct {
	Services map[string]struct {
		DependsOn  map[string]any `yaml:"depends_on"`
		ExtraHosts []string       `yaml:"extra_hosts"`
		Ports      []string       `yaml:"ports"`
	} `yaml:"services"`
	Volumes map[string]any `yaml:"volumes"`
}

// writeDevelopmentFiles applies every file step of a development plan. It
// returns the compose file, parsed as real YAML, and the text of odoo.conf.
func writeDevelopmentFiles(t *testing.T, req Request) (composeShape, string) {
	t.Helper()
	for _, step := range BuildPlan(req) {
		if step.Kind != KindFile && step.Kind != KindProject {
			continue
		}
		if err := Apply(context.Background(), step, req, nil); err != nil {
			t.Fatalf("Apply(%s) error = %v", step.Target, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(req.Folder(), "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var shape composeShape
	if err := yaml.Unmarshal(data, &shape); err != nil {
		t.Fatalf("the generated compose file is not valid YAML: %v\n%s", err, data)
	}
	conf, err := os.ReadFile(filepath.Join(req.Folder(), "odoo.conf"))
	if err != nil {
		t.Fatal(err)
	}
	return shape, string(conf)
}

func TestDockerStackKeepsTheDatabaseContainer(t *testing.T) {
	req := Request{Environment: "dev", Version: "20.0", Path: t.TempDir(), Port: 8069}
	compose, conf := writeDevelopmentFiles(t, req)

	if len(compose.Services) != 2 {
		t.Fatalf("services = %v, want web and db", compose.Services)
	}
	web := compose.Services["web"]
	if _, waits := web.DependsOn["db"]; !waits {
		t.Fatalf("web must wait for db: %+v", web)
	}
	if len(web.ExtraHosts) != 0 || len(web.Ports) != 1 || web.Ports[0] != "8069:8069" {
		t.Fatalf("web = %+v", web)
	}
	if len(compose.Volumes) != 2 {
		t.Fatalf("volumes = %v, want odoo-data and db-data", compose.Volumes)
	}
	if !strings.Contains(conf, "db_host = db\n") || !strings.Contains(conf, "db_password = odoo\n") {
		t.Fatalf("odoo.conf = \n%s", conf)
	}

	config, err := project.Load(req.Folder())
	if err != nil || config.Docker.DatabaseService != "db" {
		t.Fatalf("database service = %q, error = %v; want db", config.Docker.DatabaseService, err)
	}
}

func TestHostDatabaseStackLeavesTheContainerOut(t *testing.T) {
	req := Request{Environment: "dev", Version: "20.0", Path: t.TempDir(), Port: 8169,
		Stack: StackHostDB, DBUser: "shop", DBPassword: "s3cret", DBPort: 5433}
	compose, conf := writeDevelopmentFiles(t, req)

	if len(compose.Services) != 1 {
		t.Fatalf("services = %v, want only web", compose.Services)
	}
	web := compose.Services["web"]
	if len(web.DependsOn) != 0 {
		t.Fatalf("web must not wait for a database container: %+v", web)
	}
	if len(web.ExtraHosts) != 1 || web.ExtraHosts[0] != "host.docker.internal:host-gateway" {
		t.Fatalf("the Odoo container needs a way to reach this computer: %+v", web)
	}
	if len(compose.Volumes) != 1 {
		t.Fatalf("volumes = %v, want only odoo-data", compose.Volumes)
	}
	for _, want := range []string{"db_host = host.docker.internal\n", "db_port = 5433\n", "db_user = shop\n", "db_password = s3cret\n"} {
		if !strings.Contains(conf, want) {
			t.Fatalf("odoo.conf is missing %q:\n%s", want, conf)
		}
	}

	config, err := project.Load(req.Folder())
	if err != nil || config.Docker.DatabaseService != "" {
		t.Fatalf("database service = %q, error = %v; want none", config.Docker.DatabaseService, err)
	}
}

func TestProblemsChecksTheStack(t *testing.T) {
	if got := (Request{Environment: "dev", Version: "20.0", Stack: "kubernetes"}).Problems(); len(got) != 1 {
		t.Fatalf("an unknown stack must be one problem, got %v", got)
	}
	if got := (Request{Environment: "prod", Version: "20.0", Stack: StackHostDB}).Problems(); len(got) != 1 {
		t.Fatalf("host-db outside development must be one problem, got %v", got)
	}
	if got := (Request{Environment: "dev", Version: "20.0", Stack: StackHostDB}).Problems(); len(got) != 0 {
		t.Fatalf("host-db in development is fine, got %v", got)
	}
}
