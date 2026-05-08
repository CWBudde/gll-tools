package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureFileExists(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		setup   func(t *testing.T) string
		wantSub string
	}{
		{
			name:    "empty path is rejected",
			path:    "",
			wantSub: "missing file path",
		},
		{
			name:    "missing file is rejected",
			path:    filepath.Join(t.TempDir(), "does-not-exist.xgll"),
			wantSub: "file not found",
		},
		{
			name: "existing file passes",
			setup: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "ok.xgll")
				if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
				return p
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.path
			if tc.setup != nil {
				path = tc.setup(t)
			}
			err := ensureFileExists(path)
			switch {
			case tc.wantSub == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tc.wantSub != "" && err == nil:
				t.Errorf("want error containing %q, got nil", tc.wantSub)
			case tc.wantSub != "" && !strings.Contains(err.Error(), tc.wantSub):
				t.Errorf("error = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}

// TestExecute exercises the public Execute() entrypoint by setting argv to
// invoke the help subcommand (a side-effect-free path that still goes through
// the full cobra dispatch). We pre-set os.Args via the cobra rootCmd to avoid
// touching process state.
func TestExecute(t *testing.T) {
	// rootCmd is package-shared state; restore args afterwards.
	prev := os.Args
	t.Cleanup(func() { os.Args = prev })

	rootCmd.SetArgs([]string{"--help"})
	rootCmd.SetOut(&strings.Builder{})
	rootCmd.SetErr(&strings.Builder{})

	if err := Execute(); err != nil {
		t.Fatalf("Execute() = %v, want nil for --help", err)
	}
}
