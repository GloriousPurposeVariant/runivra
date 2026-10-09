package database

import "testing"

func TestParseNames(t *testing.T) {
	names := parseNames("shop_dev\r\nshop_test\n\n")
	if len(names) != 2 || names[0] != "shop_dev" || names[1] != "shop_test" {
		t.Fatalf("names = %q", names)
	}
	if len(parseNames("")) != 0 {
		t.Fatal("no output must mean no databases")
	}
}

func TestChoose(t *testing.T) {
	names := []string{"shop_dev", "shop_test"}

	if got, err := Choose(names, "shop_test", "shop_dev"); err != nil || got != "shop_test" {
		t.Fatalf("an explicit name must win: got %q, %v", got, err)
	}
	if _, err := Choose(names, "missing", ""); err == nil {
		t.Fatal("an explicit name that does not exist must be an error")
	}
	if got, _ := Choose(names, "", "shop_dev"); got != "shop_dev" {
		t.Fatalf("the remembered database must be used: got %q", got)
	}
	if got, _ := Choose(names, "", "dropped_since"); got != "" {
		t.Fatalf("a remembered database that no longer exists must not be used: got %q", got)
	}
	if got, _ := Choose([]string{"only"}, "", ""); got != "only" {
		t.Fatalf("a single database must be chosen without asking: got %q", got)
	}
	if got, err := Choose(names, "", ""); got != "" || err != nil {
		t.Fatalf("with several and nothing remembered the caller must ask: got %q, %v", got, err)
	}
}
