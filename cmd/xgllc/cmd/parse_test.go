package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/xgll"
)

func TestWriteDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostics(&buf, "Diagnostics", []xgll.Diagnostic{
		{
			Severity: xgll.SeverityWarning,
			Message:  "missing Layout block",
			Line:     4,
			Column:   1,
		},
	})

	if buf.Len() == 0 {
		t.Fatalf("expected output")
	}
}

func TestParseCommand(t *testing.T) {
	var buf bytes.Buffer

	path := filepath.FromSlash("../../../testdata/xgll/example-ls.xgll")
	cmd := parseCmd
	cmd.SetArgs([]string{path})
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("parse command failed: %v", err)
	}
}

// TestParseCommandJSONOutput exercises the --json branch (writeJSON is at 0%
// without this). We pin both rootCmd and parseCmd output writers because
// earlier tests in the package directly invoke parseCmd.Execute() and leave
// stale writer state on the subcommand.
func TestParseCommandJSONOutput(t *testing.T) {
	var buf bytes.Buffer

	path := filepath.FromSlash("../../../testdata/xgll/example-ls.xgll")
	jsonOut = true
	t.Cleanup(func() { jsonOut = false })

	rootCmd.SetArgs([]string{"parse", path, "--json"})
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	parseCmd.SetOut(&buf)
	parseCmd.SetErr(&buf)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("parse --json failed: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("expected JSON object output, got: %q", out)
	}
	// Sanity-check the JSON includes the document fields we expect.
	for _, want := range []string{"\"Statements\"", "\"Blocks\""} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON output missing %q", want)
		}
	}
}

func TestParseCommandMissingFile(t *testing.T) {
	rootCmd.SetArgs([]string{"parse", filepath.Join(t.TempDir(), "missing.xgll")})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestWriteJSONDirect(t *testing.T) {
	var buf bytes.Buffer
	if err := writeJSON(&buf, map[string]int{"a": 1, "b": 2}); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\"a\": 1") || !strings.Contains(out, "\"b\": 2") {
		t.Errorf("writeJSON output missing keys: %s", out)
	}
}
