package xgll

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"
)

func TestXGLLBinWriter(t *testing.T) {
	doc, err := ParseFile(filepath.FromSlash("../../testdata/xgll/example-ls.xgll"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	writer, err := GetWriter("xgllbin")
	if err != nil {
		t.Fatalf("get writer failed: %v", err)
	}

	var buf bytes.Buffer
	if err := writer.Write(doc, &buf); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if buf.Len() < len(xgllbinMagic)+6 {
		t.Fatalf("unexpected output size: %d", buf.Len())
	}

	magic := make([]byte, len(xgllbinMagic))
	if _, err := buf.Read(magic); err != nil {
		t.Fatalf("read magic: %v", err)
	}

	if string(magic) != xgllbinMagic {
		t.Fatalf("unexpected magic: %q", string(magic))
	}

	var version uint16
	if err := binary.Read(&buf, binary.LittleEndian, &version); err != nil {
		t.Fatalf("read version: %v", err)
	}

	if version != xgllbinVersion {
		t.Fatalf("unexpected version: %d", version)
	}

	var length uint32
	if err := binary.Read(&buf, binary.LittleEndian, &length); err != nil {
		t.Fatalf("read length: %v", err)
	}

	if int(length) != buf.Len() {
		t.Fatalf("length mismatch: header %d, payload %d", length, buf.Len())
	}
}

func TestXGLLBinPrettyWriter(t *testing.T) {
	doc, err := ParseFile(filepath.FromSlash("../../testdata/xgll/example-ls.xgll"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	writer, err := GetWriter("xgllbin-pretty")
	if err != nil {
		t.Fatalf("get writer failed: %v", err)
	}

	var buf bytes.Buffer
	if err := writer.Write(doc, &buf); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	header := make([]byte, len(xgllbinMagic)+2+4)
	if _, err := buf.Read(header); err != nil {
		t.Fatalf("read header: %v", err)
	}

	payload := buf.String()
	if !strings.Contains(payload, "\n  ") {
		t.Fatalf("expected pretty-printed payload")
	}
}
