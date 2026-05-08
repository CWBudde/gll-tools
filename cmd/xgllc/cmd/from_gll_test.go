package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFromGLLCommand_FileOutput converts a real GLL fixture to XGLL and
// verifies the resulting file is non-empty and starts with the XGLL preamble.
func TestFromGLLCommand_FileOutput(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "from.xgll")
	in := filepath.FromSlash("../../../testdata/gll/D12-v10.gll")

	rootCmd.SetArgs([]string{"from-gll", in, "--output", out})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("from-gll: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("output file is empty")
	}
	// XGLL files begin with a "GLL" sentinel line.
	if !strings.HasPrefix(string(data), "\"GLL\"") {
		t.Errorf("output does not start with XGLL sentinel: %q", string(data[:min(len(data), 40)]))
	}
}

func TestFromGLLCommand_MissingFile(t *testing.T) {
	rootCmd.SetArgs([]string{"from-gll", filepath.Join(t.TempDir(), "missing.gll")})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestValidateCommandJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	in := filepath.FromSlash("../../../testdata/xgll/example-la.xgll")

	jsonOut = true
	t.Cleanup(func() { jsonOut = false })

	rootCmd.SetArgs([]string{"validate", in, "--json"})
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// validate may legitimately return an error when issues are detected; we
	// only care that the JSON payload was emitted.
	_ = rootCmd.Execute()

	out := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("expected JSON object output, got: %q", out)
	}
	// The wrapper keys are explicitly snake_case (see validate.go).
	for _, want := range []string{"\"document\"", "\"system_diagnostics\"", "\"data_diagnostics\""} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON output missing %q", want)
		}
	}
}
