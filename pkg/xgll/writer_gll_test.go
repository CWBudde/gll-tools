package xgll

import (
	"bytes"
	"path/filepath"
	"testing"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

func TestGLLWriterRoundTripHeader(t *testing.T) {
	// Parse XGLL example
	doc, err := ParseFile(filepath.FromSlash("../../testdata/xgll/example-ls.xgll"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Get GLL writer
	writer, err := GetWriter("gll")
	if err != nil {
		t.Fatalf("get writer failed: %v", err)
	}

	// Write GLL into buffer
	var buf bytes.Buffer
	if err := writer.Write(doc, &buf); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Parse back into GLL structure
	reader := bytes.NewReader(buf.Bytes())

	file, err := gllbin.Parse(reader)
	if err != nil {
		t.Fatalf("parse gll failed: %v", err)
	}

	// Basic header fields should be populated
	if file.GenSystem.Label == "" {
		t.Fatalf("expected label")
	}

	// Key should also be present
	if file.GenSystem.Key == "" {
		t.Fatalf("expected key")
	}
}
