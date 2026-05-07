package cmd

import (
	"path/filepath"
	"testing"
)

func TestAcousticCommandAll(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", path); err != nil {
		t.Fatalf("acoustic command failed: %v", err)
	}
}

func TestAcousticCommandSpecificSource(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", "--source", "0", path); err != nil {
		t.Fatalf("acoustic --source 0 command failed: %v", err)
	}
}

func TestAcousticCommandSourceOutOfRange(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", "--source", "999", path); err == nil {
		t.Fatal("expected error for out-of-range source index")
	}
}

func TestAcousticCommandResponses(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", "--source", "0", "--responses", "--max-responses", "2", path); err != nil {
		t.Fatalf("acoustic --responses command failed: %v", err)
	}
}

func TestAcousticCommandExportCSV(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "responses.csv")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", "--source", "0", "--export-csv", out, path); err != nil {
		t.Fatalf("acoustic --export-csv command failed: %v", err)
	}
}

func TestAcousticCommandExportFRD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", "--source", "0", "--export-frd", dir, path); err != nil {
		t.Fatalf("acoustic --export-frd command failed: %v", err)
	}
}

func TestAcousticCommandExportCLF(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.clf")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", "--source", "0", "--export-clf", out, path); err != nil {
		t.Fatalf("acoustic --export-clf command failed: %v", err)
	}
}

func TestAcousticCommandCSVRequiresSource(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "responses.csv")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", "--export-csv", out, path); err == nil {
		t.Fatal("expected error: --export-csv requires --source")
	}
}

func TestAcousticCommandCLFRequiresSource(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.clf")
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", "--export-clf", out, path); err == nil {
		t.Fatal("expected error: --export-clf requires --source")
	}
}

func TestAcousticCommandFRDRequiresSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "acoustic", "--export-frd", dir, path); err == nil {
		t.Fatal("expected error: --export-frd requires --source")
	}
}

func TestAcousticCommandMissingFile(t *testing.T) {
	if err := runRoot(t, "acoustic", "nonexistent.gll"); err == nil {
		t.Fatal("expected error for missing file")
	}
}
