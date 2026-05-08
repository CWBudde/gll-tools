package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertCommandXGLLBin(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.xgllbin")
	path := filepath.FromSlash("../../../testdata/xgll/example-ls.xgll")

	// Reset flags to default values
	convertOutput = ""
	convertFormat = "xgllbin"
	convertPretty = false

	// Execute through root command with subcommand
	rootCmd.SetArgs([]string{"convert", path, "--output", output, "--format", "xgllbin"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert command failed: %v", err)
	}

	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("output missing: %v", err)
	}

	if info.Size() == 0 {
		t.Fatalf("output is empty")
	}
}

// TestConvertCommandErrors covers the four error paths: missing file, missing
// --output, --pretty without xgllbin, and an unknown format.
func TestConvertCommandErrors(t *testing.T) {
	validInput := filepath.FromSlash("../../../testdata/xgll/example-ls.xgll")
	dir := t.TempDir()

	tests := []struct {
		name    string
		args    []string
		wantSub string
	}{
		{
			name:    "missing file",
			args:    []string{"convert", filepath.Join(dir, "missing.xgll"), "--output", filepath.Join(dir, "x.xgllbin")},
			wantSub: "file not found",
		},
		{
			name:    "missing output",
			args:    []string{"convert", validInput},
			wantSub: "missing --output",
		},
		{
			name:    "pretty without xgllbin",
			args:    []string{"convert", validInput, "--output", filepath.Join(dir, "x.gll"), "--format", "gll", "--pretty"},
			wantSub: "--pretty is only supported",
		},
		{
			name:    "unknown format",
			args:    []string{"convert", validInput, "--output", filepath.Join(dir, "x.bogus"), "--format", "no-such-format"},
			wantSub: "supported:",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset shared state between cases.
			convertOutput = ""
			convertFormat = "xgllbin"
			convertPretty = false

			var buf bytes.Buffer
			rootCmd.SetArgs(tc.args)
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			err := rootCmd.Execute()
			if err == nil {
				t.Fatalf("want error containing %q, got nil", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestConvertCommandPretty(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out-pretty.xgllbin")
	path := filepath.FromSlash("../../../testdata/xgll/example-ls.xgll")

	// Reset flags to default values
	convertOutput = ""
	convertFormat = "xgllbin"
	convertPretty = false

	// Execute through root command with subcommand
	rootCmd.SetArgs([]string{"convert", path, "--output", output, "--format", "xgllbin", "--pretty"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("convert command failed: %v", err)
	}

	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("output missing: %v", err)
	}

	if info.Size() == 0 {
		t.Fatalf("output is empty")
	}
}
