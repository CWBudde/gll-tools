package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCommandExample(t *testing.T) {
	path := filepath.FromSlash("../../../testdata/xgll/example-la.xgll")

	rootCmd.SetArgs([]string{"validate", path})
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("validate command failed: %v", err)
	}
}

func TestValidateCommandError(t *testing.T) {
	// LA system without required Frames and Connectors blocks
	input := "\"GLL\"\n\"Format\", \"3D\"\n\"FormatVersion\", \"1.0\"\n\"System\", \"LA\", \"sys\", \"LA\"\n\"Data\"\n"

	tmp, err := os.CreateTemp("", "bad-*.xgll")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(input); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := tmp.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	rootCmd.SetArgs([]string{"validate", tmp.Name()})
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected validation error")
	}
}
