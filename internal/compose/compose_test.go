package compose

import "testing"

const sample = `{
  "services": {
    "db": {"image": "postgres:16"},
    "web": {"ports": [
      {"mode": "ingress", "target": 8072, "published": "8172", "protocol": "tcp"},
      {"mode": "ingress", "target": 8069, "published": "8169", "protocol": "tcp"}
    ]}
  }
}`

func TestParsePublishedPort(t *testing.T) {
	port, err := parsePublishedPort([]byte(sample), "web", 8069)
	if err != nil || port != 8169 {
		t.Fatalf("port = %d, error = %v; want 8169", port, err)
	}
	if _, err := parsePublishedPort([]byte(sample), "db", 8069); err == nil {
		t.Fatal("a service that publishes nothing must return an error")
	}
	if _, err := parsePublishedPort([]byte(sample), "odoo", 8069); err == nil {
		t.Fatal("a service that does not exist must return an error")
	}
}
