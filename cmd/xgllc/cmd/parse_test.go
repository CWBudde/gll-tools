package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/MeKo-Christian/gll-tools/pkg/xgll"
)

func TestWriteDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	writeDiagnostics(&buf, "Diagnostics", []xgll.Diagnostic{
		{
			Severity: xgll.SeverityWarning,
			Message:  "missing Layout block",
			Line:     4,
			Column:   1,
		},
	})

	if buf.Len() == 0 {
		t.Fatalf("expected output")
	}
}

func TestParseCommand(t *testing.T) {
	var buf bytes.Buffer

	path := filepath.FromSlash("../../../testdata/xgll/example-ls.xgll")
	cmd := parseCmd
	cmd.SetArgs([]string{path})
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("parse command failed: %v", err)
	}
}
