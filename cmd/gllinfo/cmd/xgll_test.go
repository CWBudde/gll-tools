package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestXGLLCommand_StdoutDefault drives the no-output-flag branch of runXGLL,
// which writes to os.Stdout. We swap os.Stdout for a pipe and read it in a
// goroutine to avoid blocking the writer.
func TestXGLLCommand_StdoutDefault(t *testing.T) {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	// Drain the pipe in the background so writes never block.
	captured := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		captured <- buf.Bytes()
	}()

	viper.Set("xgll.output", "")

	in := filepath.FromSlash("../../../testdata/gll/D12-v10.gll")
	rootCmd.SetArgs([]string{"xgll", in})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	execErr := rootCmd.Execute()

	// Close the writer so the goroutine sees EOF and returns.
	_ = w.Close()
	out := string(<-captured)

	if execErr != nil {
		t.Fatalf("xgll command: %v", execErr)
	}
	if !strings.HasPrefix(out, "\"GLL\"") {
		t.Errorf("output does not start with XGLL sentinel: %q", out[:min(len(out), 40)])
	}
}

func TestXGLLCommand_FileOutput(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "from.xgll")
	in := filepath.FromSlash("../../../testdata/gll/D12-v10.gll")

	viper.Set("xgll.output", out)
	t.Cleanup(func() { viper.Set("xgll.output", "") })

	rootCmd.SetArgs([]string{"xgll", in, "--output", out})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("xgll command: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.HasPrefix(string(data), "\"GLL\"") {
		t.Errorf("output does not start with XGLL sentinel: %q", data[:min(len(data), 40)])
	}
}

func TestXGLLCommand_MissingFile(t *testing.T) {
	viper.Set("xgll.output", "")
	rootCmd.SetArgs([]string{"xgll", filepath.Join(t.TempDir(), "missing.gll")})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestExecute exercises the public Execute() wrapper using --help (a
// side-effect-free invocation through the full cobra dispatch).
func TestExecute(t *testing.T) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	rootCmd.SetArgs([]string{"--help"})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	if err := Execute(); err != nil {
		t.Fatalf("Execute() = %v, want nil for --help", err)
	}
}
