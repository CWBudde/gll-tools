package xgll

import (
	"bytes"
	"math"
	"path/filepath"
	"testing"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

func TestGLLWriterSourceDefinitionRoundTrip(t *testing.T) {
	const spl = 90.0
	src := SyntheticSource("Full Range", "sdTest", spl)

	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = 4
	file.GenSystem.Label = "Roundtrip Test"
	file.GenSystem.Key = "sysTest"
	file.GenSystem.Type = gllbin.SystemTypeLoudspeaker
	file.Database = &gllbin.Database{
		SubVersion:        3,
		SourceDefinitions: []gllbin.SourceDefinitionItem{src},
	}

	var buf bytes.Buffer
	if err := EncodeFile(file, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	data := buf.Bytes()
	parsed, err := gllbin.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if parsed.Database == nil {
		t.Fatal("database is nil")
	}
	if len(parsed.Database.SourceDefinitions) != 1 {
		t.Fatalf("want 1 source definition, got %d", len(parsed.Database.SourceDefinitions))
	}

	item := parsed.Database.SourceDefinitions[0]
	if item.Key != "sdTest" {
		t.Errorf("key: want sdTest, got %q", item.Key)
	}

	def := item.Definition
	if def == nil {
		t.Fatal("definition is nil")
	}
	if def.Label != "Full Range" {
		t.Errorf("label: want Full Range, got %q", def.Label)
	}

	balloon := def.BalloonData
	if balloon == nil {
		t.Fatal("balloon is nil")
	}
	wantResponses := src.Definition.BalloonData.AngularResolution.ParallelCount()
	if int(balloon.ResponseCount) != wantResponses {
		t.Errorf("response count: want %d, got %d", wantResponses, balloon.ResponseCount)
	}

	// Load balloon responses lazily (requires the raw bytes as a seeker).
	if err := gllbin.LoadBalloonResponses(bytes.NewReader(data), balloon); err != nil {
		t.Fatalf("load balloon responses: %v", err)
	}
	if len(balloon.Responses) != wantResponses {
		t.Fatalf("loaded %d responses, want %d", len(balloon.Responses), wantResponses)
	}

	// Verify level values round-trip through int16 scaling.
	const tol = 0.01 // 1 LSB in the 0.01 dB scale
	for i, resp := range balloon.Responses {
		for j, lv := range resp.Level {
			if math.Abs(lv-spl) > tol {
				t.Errorf("response[%d].Level[%d]: want %.2f, got %.2f", i, j, spl, lv)
			}
		}
	}
}

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
