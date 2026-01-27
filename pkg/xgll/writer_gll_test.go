package xgll

import (
	"bytes"
	"path/filepath"
	"testing"

	gllbin "github.com/MeKo-Christian/gll-tools/pkg/gll"
)

func TestGLLWriterRoundTripHeader(t *testing.T) {
	doc, err := ParseFile(filepath.FromSlash("../../testdata/xgll/example-ls.xgll"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	writer, err := GetWriter("gll")
	if err != nil {
		t.Fatalf("get writer failed: %v", err)
	}

	var buf bytes.Buffer
	if err := writer.Write(doc, &buf); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	reader := bytes.NewReader(buf.Bytes())

	file, err := gllbin.Parse(reader)
	if err != nil {
		t.Fatalf("parse gll failed: %v", err)
	}

	if file.GenSystem.Label == "" {
		t.Fatalf("expected label")
	}

	if file.GenSystem.Key == "" {
		t.Fatalf("expected key")
	}
}
