package setup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/project"
)

func hostRequest(t *testing.T) Request {
	t.Helper()
	return Request{Environment: "dev", Version: "17.0", Path: t.TempDir(), Port: 8170,
		Stack: StackHost, DBUser: "odoo_dev", DBPassword: "s3cret", DBPort: 5432}
}

func TestHostPlanHasNoDockerFiles(t *testing.T) {
	req := hostRequest(t)
	var kinds []string
	for _, step := range BuildPlan(req) {
		kinds = append(kinds, step.Kind)
		name := filepath.Base(step.Target)
		if name == "Dockerfile" || name == "docker-compose.yml" || name == ".dockerignore" {
			t.Fatalf("a host project must not get %s", name)
		}
	}
	want := []string{KindClone, KindFolder, KindFolder, KindVenv, KindPackages, KindFile, KindProject}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("steps = %v, want %v", kinds, want)
	}
}

func TestHostOdooConfUsesRealPaths(t *testing.T) {
	req := hostRequest(t)
	for _, step := range BuildPlan(req) {
		if step.Kind != KindFile && step.Kind != KindProject {
			continue
		}
		if err := Apply(context.Background(), step, req, nil); err != nil {
			t.Fatalf("Apply(%s) error = %v", step.Target, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(req.Folder(), "odoo.conf"))
	if err != nil {
		t.Fatal(err)
	}
	conf := string(data)
	base, _ := filepath.Abs(req.Folder())
	for _, want := range []string{
		"addons_path = " + filepath.Join(base, "custom") + "," + filepath.Join(base, "enterprise") + "," + filepath.Join(base, "addons") + "\n",
		"db_host = localhost\n",
		"db_user = odoo_dev\n",
		"db_password = s3cret\n",
		"http_port = 8170\n",
	} {
		if !strings.Contains(conf, want) {
			t.Fatalf("odoo.conf is missing %q:\n%s", want, conf)
		}
	}
	if strings.Contains(conf, "host.docker.internal") || strings.Contains(conf, "/opt/odoo") {
		t.Fatalf("a host odoo.conf must not contain container names or paths:\n%s", conf)
	}

	config, err := project.Load(req.Folder())
	if err != nil {
		t.Fatal(err)
	}
	if config.Odoo.Runtime != "host" || config.Paths.Python != ".venv" || config.Docker.ComposeFile != "" {
		t.Fatalf("config = %+v; want a host project with no compose file", config)
	}
}

func TestHostStackIsOnlyForDevelopment(t *testing.T) {
	if got := (Request{Environment: "prod", Version: "17.0", Stack: StackHost}).Problems(); len(got) != 1 {
		t.Fatalf("host outside development must be one problem, got %v", got)
	}
	req := Request{Environment: "dev", Version: "17.0", Stack: StackHost}
	if !req.HostOdoo() || !req.HostDatabase() {
		t.Fatal("the host stack runs both Odoo and the database on this computer")
	}
	if (Request{Stack: StackHostDB}).HostOdoo() {
		t.Fatal("host-db keeps Odoo in Docker")
	}
}

func TestWarningIsAnError(t *testing.T) {
	var err error = &Warning{Summary: "1 package could not be installed", Details: []string{"libsass"}}
	if err.Error() != "1 package could not be installed" {
		t.Fatalf("Error() = %q", err.Error())
	}
}
