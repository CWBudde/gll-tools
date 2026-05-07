package cmd

import (
	"path/filepath"
	"testing"
)

func TestGetExtensionForContent(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"pdf-cmap", ".pdf.txt"},
		{"pdf-graphics", ".pdf.txt"},
		{"font-ttf", ".ttf"},
		{"font-data", ".otf"},
		{"text", ".txt"},
		{"acoustic-data", ".dat"},
		{"unknown", ".bin"},
		{"", ".bin"},
		// Case sensitivity: switch is exact-match, so capitalized variants
		// fall through to the default.
		{"PDF-CMAP", ".bin"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := getExtensionForContent(tc.in); got != tc.want {
				t.Errorf("getExtensionForContent(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestCleanFilename(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain name", "image.png", "image.png"},
		{"windows backslash", `C:\path\to\file.png`, "file.png"},
		{"unix path", "/var/tmp/data.bin", "data.bin"},
		{"leading dot-slash", "./local.dat", "local.dat"},
		{"mixed separators", `dir\sub/file.txt`, "file.txt"},
		{"already base", "base", "base"},
		{"empty stays empty", "", "."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanFilename(tc.in); got != tc.want {
				t.Errorf("cleanFilename(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestExtractCommandNoDecompress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.FromSlash(testGLLFile)
	// Exercises processZlibResource's non-decompress branch (raw .zlib output).
	if err := runRoot(t, "extract", "--output", dir, "--decompress=false", path); err != nil {
		t.Fatalf("extract --decompress=false failed: %v", err)
	}
}

func TestExtractCommandImagesOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "extract", "--output", dir, "--images", path); err != nil {
		t.Fatalf("extract --images failed: %v", err)
	}
}

func TestExtractCommandDataOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "extract", "--output", dir, "--data", path); err != nil {
		t.Fatalf("extract --data failed: %v", err)
	}
}

func TestExtractCommandDocsOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "extract", "--output", dir, "--docs", path); err != nil {
		t.Fatalf("extract --docs failed: %v", err)
	}
}

func TestExtractCommandMissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := runRoot(t, "extract", "--output", dir, "nonexistent.gll"); err == nil {
		t.Fatal("expected error for missing file")
	}
}
