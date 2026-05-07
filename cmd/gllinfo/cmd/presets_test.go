package cmd

import (
	"path/filepath"
	"testing"
)

func TestPresetsCommand(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "presets", path); err != nil {
		t.Fatalf("presets command failed: %v", err)
	}
}

func TestPresetsCommandJSON(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "presets", "--json", path); err != nil {
		t.Fatalf("presets --json command failed: %v", err)
	}
}

func TestPresetsCommandDecode(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "presets", "--decode", path); err != nil {
		t.Fatalf("presets --decode command failed: %v", err)
	}
}

func TestPresetsCommandJSONRaw(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "presets", "--json", "--raw", path); err != nil {
		t.Fatalf("presets --json --raw command failed: %v", err)
	}
}

func TestPresetsCommandMissingFile(t *testing.T) {
	if err := runRoot(t, "presets", "nonexistent.gll"); err == nil {
		t.Fatal("expected error for missing file")
	}
}
