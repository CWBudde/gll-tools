package xgll

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExamples(t *testing.T) {
	// Parse known example files
	files := []string{
		"../../testdata/xgll/example-ls.xgll",
		"../../testdata/xgll/example-la.xgll",
		"../../testdata/xgll/example-cl.xgll",
	}

	// Validate each input parses into statements/blocks
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			doc, err := ParseFile(filepath.FromSlash(file))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			if len(doc.Statements) == 0 {
				t.Fatalf("no statements parsed")
			}

			// Expect block generation
			if len(doc.Blocks) == 0 {
				t.Fatalf("no blocks generated")
			}
		})
	}
}

func TestParseInvalidString(t *testing.T) {
	// Unterminated string should fail
	input := "\"GLL\"\n\"Format\", \"3D\"\n\"FormatVersion\", \"1.0\"\n\"System\", \"Bad\n"

	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatalf("expected error for unterminated string")
	}
}

func TestValidateBlockOrder(t *testing.T) {
	// Layout must precede Data in block order
	input := strings.Join([]string{
		"\"GLL\"",
		"\"Format\", \"3D\"",
		"\"FormatVersion\", \"1.0\"",
		"\"System\", \"Example\", \"sys\", \"LS\"",
		"\"Data\"",
		"\"Layout\"",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatalf("expected error for invalid block order")
	}

	// Expect diagnostics on invalid order
	if doc == nil {
		t.Fatalf("expected document with diagnostics")
	}
}
