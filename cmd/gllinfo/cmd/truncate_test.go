package cmd

import "testing"

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("truncate short = %q", got)
	}

	if got := truncate("exact", 5); got != "exact" {
		t.Fatalf("truncate exact = %q", got)
	}

	if got := truncate("this is long", 7); got != "this..." {
		t.Fatalf("truncate long = %q", got)
	}
}
