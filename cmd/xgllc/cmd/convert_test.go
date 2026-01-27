package cmd

import (
	"os"
	"path/filepath"
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
