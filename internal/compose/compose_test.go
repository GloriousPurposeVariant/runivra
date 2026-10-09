package compose

import "testing"

const sample = `{
  "services": {
    "db": {"image": "postgres:16"},
    "web": {
      "build": {"context": "/srv/shop", "dockerfile": "Dockerfile"},
      "ports": [
        {"mode": "ingress", "target": 8072, "published": "8172", "protocol": "tcp"},
        {"mode": "ingress", "target": 8069, "published": "8169", "protocol": "tcp"}
      ],
      "volumes": [
        {"type": "bind", "source": "/srv/shop/custom", "target": "/opt/odoo/custom"},
        {"type": "volume", "source": "odoo-data", "target": "/var/lib/odoo"}
      ]
    }
  }
}`

func TestParse(t *testing.T) {
	file, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	web := file.Services["web"]
	if web.Build == nil || web.Build.Context != "/srv/shop" {
		t.Fatalf("build = %+v, want the build context", web.Build)
	}
	if len(web.Volumes) != 2 || web.Volumes[0].Type != "bind" || web.Volumes[0].Target != "/opt/odoo/custom" {
		t.Fatalf("volumes = %+v", web.Volumes)
	}
	if file.Services["db"].Image != "postgres:16" || file.Services["db"].Build != nil {
		t.Fatalf("db = %+v", file.Services["db"])
	}
}

func TestPublishedPort(t *testing.T) {
	file, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	port, err := file.PublishedPort("web", 8069)
	if err != nil || port != 8169 {
		t.Fatalf("port = %d, error = %v; want 8169", port, err)
	}
	if _, err := file.PublishedPort("db", 8069); err == nil {
		t.Fatal("a service that publishes nothing must return an error")
	}
	if _, err := file.PublishedPort("odoo", 8069); err == nil {
		t.Fatal("a service that does not exist must return an error")
	}
}
