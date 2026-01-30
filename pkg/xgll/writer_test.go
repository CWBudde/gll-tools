package xgll

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"
)

func TestXGLLBinWriter(t *testing.T) {
	// Parse XGLL example
	doc, err := ParseFile(filepath.FromSlash("../../testdata/xgll/example-ls.xgll"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Get binary writer
	writer, err := GetWriter("xgllbin")
	if err != nil {
		t.Fatalf("get writer failed: %v", err)
	}

	// Write binary output
	var buf bytes.Buffer
	if err := writer.Write(doc, &buf); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Check minimum size for header
	if buf.Len() < len(xgllbinMagic)+6 {
		t.Fatalf("unexpected output size: %d", buf.Len())
	}

	// Read magic header
	magic := make([]byte, len(xgllbinMagic))
	if _, err := buf.Read(magic); err != nil {
		t.Fatalf("read magic: %v", err)
	}

	// Verify magic string
	if string(magic) != xgllbinMagic {
		t.Fatalf("unexpected magic: %q", string(magic))
	}

	// Read and verify version
	var version uint16
	if err := binary.Read(&buf, binary.LittleEndian, &version); err != nil {
		t.Fatalf("read version: %v", err)
	}

	if version != xgllbinVersion {
		t.Fatalf("unexpected version: %d", version)
	}

	// Read and verify payload length
	var length uint32
	if err := binary.Read(&buf, binary.LittleEndian, &length); err != nil {
		t.Fatalf("read length: %v", err)
	}

	if int(length) != buf.Len() {
		t.Fatalf("length mismatch: header %d, payload %d", length, buf.Len())
	}
}

func TestXGLLBinPrettyWriter(t *testing.T) {
	// Parse XGLL example
	doc, err := ParseFile(filepath.FromSlash("../../testdata/xgll/example-ls.xgll"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Get pretty binary writer
	writer, err := GetWriter("xgllbin-pretty")
	if err != nil {
		t.Fatalf("get writer failed: %v", err)
	}

	// Write pretty binary output
	var buf bytes.Buffer
	if err := writer.Write(doc, &buf); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Skip header bytes
	header := make([]byte, len(xgllbinMagic)+2+4)
	if _, err := buf.Read(header); err != nil {
		t.Fatalf("read header: %v", err)
	}

	// Ensure pretty formatting
	payload := buf.String()
	if !strings.Contains(payload, "\n  ") {
		t.Fatalf("expected pretty-printed payload")
	}
}
