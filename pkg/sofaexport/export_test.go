package sofaexport

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPattern(t *testing.T) {
	tests := []struct {
		pattern, gll, src, uc, label, want string
	}{
		{"{gll}__{source}__{usecase}.sofa", "3Way-LR", "lf", "default", "", "3Way-LR__lf__default.sofa"},
		{"{gll}__{source}.sofa", "MyGLL", "", "x", "Mid Driver", "MyGLL__Mid_Driver.sofa"},
		{"{gll}_{usecase}", "g", "s", "high power", "", "g_high_power.sofa"}, // missing .sofa appended
		{"name with spaces", "g", "s", "u", "", "name with spaces.sofa"},
	}
	for _, tt := range tests {
		got := renderPattern(tt.pattern, tt.gll, tt.src, tt.uc, tt.label)
		if got != tt.want {
			t.Errorf("renderPattern(%q,%q,%q,%q,%q) = %q, want %q",
				tt.pattern, tt.gll, tt.src, tt.uc, tt.label, got, tt.want)
		}
	}
}

func TestSanitizeFilenamePart(t *testing.T) {
	tests := map[string]string{
		"":              "default",
		"hello":         "hello",
		"a/b\\c d":      "a_b_c_d",
		"3Way-LR.gll":   "3Way-LR.gll",
		"key:with*bad?": "key_with_bad_",
	}
	for in, want := range tests {
		if got := sanitizeFilenamePart(in); got != want {
			t.Errorf("sanitizeFilenamePart(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFilenameStem(t *testing.T) {
	tests := map[string]string{
		"speaker.gll":            "speaker",
		"path/to/3Way-LR.gll":    "3Way-LR",
		"./testdata/gll/Foo.GLL": "Foo",
		"no_extension":           "no_extension",
	}
	for in, want := range tests {
		if got := filenameStem(in); got != want {
			t.Errorf("filenameStem(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestExportFile_3WayLR is an integration test: parse a real fixture, export
// to a temp dir, and assert files were produced.
func TestExportFile_3WayLR(t *testing.T) {
	src := filepath.Join("..", "..", "testdata", "gll", "3Way-LR.gll")
	out := t.TempDir()
	paths, err := ExportFile(src, Options{OutputDir: out})
	if err != nil {
		t.Fatalf("ExportFile: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no SOFA files produced")
	}
	for _, p := range paths {
		if !strings.HasSuffix(p, ".sofa") {
			t.Errorf("output %q does not end in .sofa", p)
		}
	}
}
