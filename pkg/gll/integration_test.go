package gll

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLegacyViewerFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping legacy viewer files in short mode")
	}

	dir := filepath.Clean(filepath.Join("..", "..", "legacy", "viewer"))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("legacy viewer directory not available: %v", err)
	}

	var gllFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".gll" {
			gllFiles = append(gllFiles, filepath.Join(dir, entry.Name()))
		}
	}

	if len(gllFiles) == 0 {
		t.Skip("no .gll files found in legacy viewer directory")
	}

	for _, path := range gllFiles {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("open failed: %v", err)
			}
			defer f.Close()

			parsed, err := Parse(f)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			if parsed.Header.Magic != "EGLL" {
				t.Fatalf("unexpected magic: %q", parsed.Header.Magic)
			}

			if parsed.GenSystem.Label == "" {
				t.Fatalf("missing GenSystem label")
			}
		})
	}
}
