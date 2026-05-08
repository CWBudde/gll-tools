package xgll

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestListWriterFormats covers the registry listing helper. The xgll text,
// xgllbin, and xgllbin-pretty writers register themselves via init() so this
// must always include all three.
func TestListWriterFormats(t *testing.T) {
	formats := ListWriterFormats()
	want := map[string]bool{"xgll": false, "xgllbin": false, "xgllbin-pretty": false}
	for _, f := range formats {
		if _, ok := want[f]; ok {
			want[f] = true
		}
	}
	for k, v := range want {
		if !v {
			t.Errorf("ListWriterFormats() missing %q; got %v", k, formats)
		}
	}
}

// TestGetWriterUnknown ensures GetWriter surfaces an unknown-format error.
func TestGetWriterUnknown(t *testing.T) {
	_, err := GetWriter("definitely-not-a-format")
	if err == nil {
		t.Fatal("want error for unknown format, got nil")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("error = %q, want substring 'unknown format'", err.Error())
	}
}

// TestXGLLTextWriter_RoundTripStatements parses an XGLL example, writes it
// back through xgllTextWriter, and verifies the output is a non-empty XGLL
// document beginning with the "GLL" sentinel and re-parseable.
func TestXGLLTextWriter_RoundTripStatements(t *testing.T) {
	doc, err := ParseFile(filepath.FromSlash("../../testdata/xgll/example-ls.xgll"))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	w, err := GetWriter("xgll")
	if err != nil {
		t.Fatalf("GetWriter(xgll): %v", err)
	}

	var buf bytes.Buffer
	if err := w.Write(doc, &buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "\"GLL\"") {
		t.Errorf("output does not start with XGLL sentinel: %q", out[:min(len(out), 40)])
	}

	// Round-trip: re-parse the emitted text.
	roundDoc, err := Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse failed: %v", err)
	}
	if len(roundDoc.Statements) == 0 {
		t.Errorf("round-tripped doc has no statements")
	}
}

// TestWriteXGLL_NilDocument verifies the validation guard.
func TestWriteXGLL_NilDocument(t *testing.T) {
	if err := WriteXGLL(nil, &bytes.Buffer{}); err == nil {
		t.Fatal("WriteXGLL(nil) = nil, want error")
	}
}

// TestSanitizeXGLLString covers the character-replacement helper used by the
// writer (quote → apostrophe, non-printable → '?').
func TestSanitizeXGLLString(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"plain", "plain"},
		{"with \"quote\"", "with 'quote'"},
		{"tab\there", "tab?here"},
		// ÿ is U+00FF — two bytes in UTF-8 (0xC3 0xBF), each replaced.
		{"highÿbyte", "high??byte"},
	}
	for _, tc := range tests {
		if got := sanitizeXGLLString(tc.in); got != tc.want {
			t.Errorf("sanitizeXGLLString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestWriteStatement covers the two ValueKinds (string, number) plus the
// default fallback branch (unknown kind → raw token written verbatim).
func TestWriteStatement(t *testing.T) {
	const unknownKind ValueKind = 99
	stmt := Statement{
		Keyword: "Box",
		Args: []Value{
			{Kind: ValueString, Str: "label"},
			{Kind: ValueNumber, Num: 42.5},
			{Kind: unknownKind, Raw: "raw-token"},
		},
	}
	var buf bytes.Buffer
	if err := writeStatement(&buf, stmt); err != nil {
		t.Fatalf("writeStatement: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "\"Box\"") {
		t.Errorf("output should start with quoted keyword: %q", out)
	}
	if !strings.Contains(out, "\"label\"") {
		t.Errorf("string arg missing: %q", out)
	}
	if !strings.Contains(out, "42.5") {
		t.Errorf("number arg missing: %q", out)
	}
	if !strings.Contains(out, "raw-token") {
		t.Errorf("default-branch raw arg missing: %q", out)
	}
}

// TestDecodeHexBytes covers the hex helper used by the binary converter.
func TestDecodeHexBytes(t *testing.T) {
	t.Run("empty returns zero buffer", func(t *testing.T) {
		got, err := decodeHexBytes("", 4)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(got) != 4 {
			t.Errorf("len = %d, want 4", len(got))
		}
		for i, b := range got {
			if b != 0 {
				t.Errorf("byte[%d] = %x, want 0", i, b)
			}
		}
	})
	t.Run("decodes valid hex", func(t *testing.T) {
		got, err := decodeHexBytes("DEADBEEF", 4)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		want := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		if !bytes.Equal(got, want) {
			t.Errorf("got %X, want %X", got, want)
		}
	})
	t.Run("rejects wrong size", func(t *testing.T) {
		_, err := decodeHexBytes("DEAD", 4)
		if err == nil || !strings.Contains(err.Error(), "expected 4 bytes") {
			t.Errorf("err = %v, want size-mismatch", err)
		}
	})
	t.Run("rejects invalid hex", func(t *testing.T) {
		_, err := decodeHexBytes("ZZZZ", 4)
		if err == nil {
			t.Fatal("want hex decode error, got nil")
		}
	})
}
