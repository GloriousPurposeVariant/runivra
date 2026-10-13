package python

import "testing"

func TestParseRequirements(t *testing.T) {
	text := `# Odoo requirements
Babel==2.9.1 ; python_version < '3.11'  # min version = 2.6.0
cryptography==3.4.8; python_version < '3.12'  # a comment

ebaysdk==2.1.5
--extra-index-url https://example.com
lxml-html-clean; python_version >= '3.12' # unpinned
`
	got := ParseRequirements(text)
	want := []string{
		"Babel==2.9.1 ; python_version < '3.11'",
		"cryptography==3.4.8; python_version < '3.12'",
		"ebaysdk==2.1.5",
		"lxml-html-clean; python_version >= '3.12'",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d requirements, want %d: %q", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("requirement %d = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestPackageName(t *testing.T) {
	cases := map[string]string{
		"Babel==2.9.1 ; python_version < '3.11'":    "Babel",
		"cryptography==3.4.8; python_version < '3'": "cryptography",
		"lxml-html-clean; python_version >= '3.12'": "lxml-html-clean",
		"pypiwin32 ; sys_platform == 'win32'":       "pypiwin32",
		"requests":                                  "requests",
	}
	for requirement, want := range cases {
		if got := packageName(requirement); got != want {
			t.Fatalf("packageName(%q) = %q, want %q", requirement, got, want)
		}
	}
}

func TestLastLine(t *testing.T) {
	output := "Collecting python-ldap\n  Building wheel\nerror: Microsoft Visual C++ 14.0 or greater is required.\nnote: This is an issue with the package\n"
	if got := lastLine(output); got != "error: Microsoft Visual C++ 14.0 or greater is required." {
		t.Fatalf("lastLine = %q, want the error line", got)
	}
	if got := lastLine("Collecting x\nSomething went wrong\n"); got != "Something went wrong" {
		t.Fatalf("lastLine = %q, want the final line when none mentions an error", got)
	}
}
