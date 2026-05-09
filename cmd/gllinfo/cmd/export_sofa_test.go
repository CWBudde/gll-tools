package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	sofa "github.com/cwbudde/go-sofa"
)

// TestExportSofaCLI_3WayLR runs `gllinfo export sofa testdata/gll/3Way-LR.gll`
// against a temp output dir and verifies the resulting files round-trip
// through go-sofa.Open with sane dimensions.
func TestExportSofaCLI_3WayLR(t *testing.T) {
	gllPath := filepath.Join("..", "..", "..", "testdata", "gll", "3Way-LR.gll")
	if _, err := os.Stat(gllPath); err != nil {
		t.Skipf("fixture %s not present: %v", gllPath, err)
	}

	out := t.TempDir()

	rootCmd.SetArgs([]string{"export", "sofa", gllPath, "-o", out})
	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatalf("read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no SOFA files produced")
	}

	for _, e := range entries {
		path := filepath.Join(out, e.Name())
		f, err := sofa.Open(path)
		if err != nil {
			t.Errorf("sofa.Open(%s): %v", path, err)
			continue
		}
		if f.DataType != "TF" {
			t.Errorf("%s: DataType = %q, want \"TF\"", e.Name(), f.DataType)
		}
		if f.SOFAConventions != "FreeFieldDirectivityTF" {
			t.Errorf("%s: SOFAConventions = %q", e.Name(), f.SOFAConventions)
		}
		if f.M <= 0 || f.N <= 0 || f.R != 1 || f.E != 1 {
			t.Errorf("%s: dims (%d,%d,%d,%d) look wrong", e.Name(), f.M, f.R, f.E, f.N)
		}
		if len(f.Frequencies) != f.N {
			t.Errorf("%s: len(Frequencies)=%d, want N=%d", e.Name(), len(f.Frequencies), f.N)
		}
		f.Close()
	}
}
