package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const testGLLFile = "../../../testdata/gll/D12-v10.gll"
const testGLLFileFilters = "../../../testdata/gll/3Way-LR.gll"

// resetFlags restores every flag in cmd and its subcommands to its default
// value. Cobra retains parsed flag values in package-level variables across
// invocations, so tests must reset them explicitly to avoid contamination.
func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

// runRoot invokes the root cobra command with the given args and captures output.
func runRoot(t *testing.T, args ...string) error {
	t.Helper()
	resetFlags(rootCmd)
	var buf bytes.Buffer
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	return rootCmd.Execute()
}

func TestInfoCommand(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, path); err != nil {
		t.Fatalf("info command failed: %v", err)
	}
}

func TestInfoCommandJSON(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "--json", path); err != nil {
		t.Fatalf("info --json command failed: %v", err)
	}
}

func TestInfoCommandMissingFile(t *testing.T) {
	err := runRoot(t, "nonexistent.gll")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestConfigCommand(t *testing.T) {
	path := filepath.FromSlash(testGLLFileFilters)
	if err := runRoot(t, "config", path); err != nil {
		t.Fatalf("config command failed: %v", err)
	}
}

func TestConfigCommandJSON(t *testing.T) {
	path := filepath.FromSlash(testGLLFileFilters)
	if err := runRoot(t, "config", "--json", path); err != nil {
		t.Fatalf("config --json command failed: %v", err)
	}
}

func TestConfigCommandFiltersOnly(t *testing.T) {
	path := filepath.FromSlash(testGLLFileFilters)
	if err := runRoot(t, "config", "--filters", path); err != nil {
		t.Fatalf("config --filters command failed: %v", err)
	}
}

func TestConfigCommandLimitsOnly(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "config", "--limits", path); err != nil {
		t.Fatalf("config --limits command failed: %v", err)
	}
}

func TestConfigCommandWarningsOnly(t *testing.T) {
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "config", "--warnings", path); err != nil {
		t.Fatalf("config --warnings command failed: %v", err)
	}
}

func TestExtractCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.FromSlash(testGLLFile)
	if err := runRoot(t, "extract", "--output", dir, path); err != nil {
		t.Fatalf("extract command failed: %v", err)
	}
}

func TestTruncateHelper(t *testing.T) {
	cases := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"", 5, ""},
		{"exact", 5, "exact"},
		{"toolong", 6, "too..."},
	}

	for _, tc := range cases {
		got := truncate(tc.input, tc.maxLen)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.want)
		}
	}
}
