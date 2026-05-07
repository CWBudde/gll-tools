package xgll

import (
	"bytes"
	"testing"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

func TestGLLEncoderGeometryRoundTrip(t *testing.T) {
	geom := &gllbin.CaseGeometry{
		IsSymmetric:  true,
		SymmetryAxis: 12.5,
		SubVersion:   1, // engages the FaceBuffer branch
		Vertices: []gllbin.Vertex{
			{Color: 0xFF0000, X: 0, Y: 0, Z: 0, Label: "v0", HasTwin: false},
			{Color: 0x00FF00, X: 1.5, Y: -2.25, Z: 3.75, Label: "v1", HasTwin: true},
			{Color: 0x0000FF, X: -4.0, Y: 5.0, Z: -6.0, Label: "", HasTwin: false},
		},
		Edges: []gllbin.Edge{
			{Color: 0x808080, V1: 0, V2: 1, Label: "e0", HasTwin: false},
			{Color: 0x202020, V1: 1, V2: 2, Label: "", HasTwin: true},
		},
		Faces: []gllbin.Face{
			{HasTwin: false, Vertices: []int32{0, 1, 2}, Color: 0xC0FFEE, Label: "tri"},
			{HasTwin: true, Vertices: []int32{2, 1, 0, -1}, Color: 0xDEADBE, Label: ""},
		},
	}

	src := SyntheticSource("Geom Source", "geomSrc", 90.0)

	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = 4
	file.GenSystem.Label = "Geometry Roundtrip"
	file.GenSystem.Key = "sysGeom"
	file.GenSystem.Type = gllbin.SystemTypeLoudspeaker
	file.Database = &gllbin.Database{
		SubVersion:        3,
		SourceDefinitions: []gllbin.SourceDefinitionItem{src},
		BoxTypes: []gllbin.BoxType{
			{
				Label:        "Box With Geom",
				Key:          "boxGeom",
				CaseGeometry: geom,
			},
		},
	}

	var buf bytes.Buffer
	if err := EncodeFile(file, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	parsed, err := gllbin.Parse(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Database == nil {
		t.Fatal("database is nil")
	}
	if len(parsed.Database.BoxTypes) != 1 {
		t.Fatalf("want 1 box type, got %d", len(parsed.Database.BoxTypes))
	}

	gotBox := parsed.Database.BoxTypes[0]
	if gotBox.Key != "boxGeom" {
		t.Errorf("box key: want boxGeom, got %q", gotBox.Key)
	}
	if gotBox.CaseGeometry == nil {
		t.Fatal("CaseGeometry not round-tripped")
	}

	got := gotBox.CaseGeometry
	if !got.IsSymmetric {
		t.Error("IsSymmetric: want true")
	}
	if got.SymmetryAxis != 12.5 {
		t.Errorf("SymmetryAxis: want 12.5, got %v", got.SymmetryAxis)
	}

	// Vertices round-trip
	if len(got.Vertices) != len(geom.Vertices) {
		t.Fatalf("vertex count: want %d, got %d", len(geom.Vertices), len(got.Vertices))
	}
	for i, want := range geom.Vertices {
		gotV := got.Vertices[i]
		if gotV.Color != want.Color || gotV.X != want.X || gotV.Y != want.Y ||
			gotV.Z != want.Z || gotV.Label != want.Label || gotV.HasTwin != want.HasTwin {
			t.Errorf("vertex[%d]: want %+v, got %+v", i, want, gotV)
		}
	}

	// Edges round-trip
	if len(got.Edges) != len(geom.Edges) {
		t.Fatalf("edge count: want %d, got %d", len(geom.Edges), len(got.Edges))
	}
	for i, want := range geom.Edges {
		gotE := got.Edges[i]
		if gotE.Color != want.Color || gotE.V1 != want.V1 || gotE.V2 != want.V2 ||
			gotE.Label != want.Label || gotE.HasTwin != want.HasTwin {
			t.Errorf("edge[%d]: want %+v, got %+v", i, want, gotE)
		}
	}

	// Faces round-trip (only if sub_version >= 1, which we set)
	if len(got.Faces) != len(geom.Faces) {
		t.Fatalf("face count: want %d, got %d", len(geom.Faces), len(got.Faces))
	}
	for i, want := range geom.Faces {
		gotF := got.Faces[i]
		if gotF.HasTwin != want.HasTwin || gotF.Color != want.Color || gotF.Label != want.Label {
			t.Errorf("face[%d] meta: want %+v, got %+v", i, want, gotF)
		}
		if len(gotF.Vertices) != len(want.Vertices) {
			t.Errorf("face[%d] vertex count: want %d, got %d",
				i, len(want.Vertices), len(gotF.Vertices))
			continue
		}
		for j := range want.Vertices {
			if gotF.Vertices[j] != want.Vertices[j] {
				t.Errorf("face[%d].Vertices[%d]: want %d, got %d",
					i, j, want.Vertices[j], gotF.Vertices[j])
			}
		}
	}
}

func TestGLLEncoderGeometryEmpty(t *testing.T) {
	// CaseGeometry with no vertices/edges/faces still encodes to a valid
	// (empty) block — exercises the empty-buffer branches in the encoders.
	geom := &gllbin.CaseGeometry{SubVersion: 1}

	src := SyntheticSource("Empty Geom Source", "esrc", 90.0)

	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = 4
	file.GenSystem.Label = "Empty Geom"
	file.GenSystem.Key = "sysEmpty"
	file.GenSystem.Type = gllbin.SystemTypeLoudspeaker
	file.Database = &gllbin.Database{
		SubVersion:        3,
		SourceDefinitions: []gllbin.SourceDefinitionItem{src},
		BoxTypes: []gllbin.BoxType{
			{Label: "Empty", Key: "boxEmpty", CaseGeometry: geom},
		},
	}

	var buf bytes.Buffer
	if err := EncodeFile(file, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	parsed, err := gllbin.Parse(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Database == nil || len(parsed.Database.BoxTypes) != 1 {
		t.Fatalf("expected 1 box, got %+v", parsed.Database)
	}
	gotBox := parsed.Database.BoxTypes[0]
	if gotBox.CaseGeometry != nil {
		if len(gotBox.CaseGeometry.Vertices) != 0 ||
			len(gotBox.CaseGeometry.Edges) != 0 ||
			len(gotBox.CaseGeometry.Faces) != 0 {
			t.Errorf("expected empty geometry buffers, got %+v", gotBox.CaseGeometry)
		}
	}
}

func TestGLLEncoderGeometrySubVersion0(t *testing.T) {
	// sub-version 0 means faces are NOT encoded — the encoder must skip the
	// FaceBuffer entirely. Round-trip should yield no faces even if we set
	// some on the input (they get dropped by design).
	geom := &gllbin.CaseGeometry{
		SubVersion: 0,
		Vertices: []gllbin.Vertex{
			{X: 0, Y: 0, Z: 0},
			{X: 1, Y: 1, Z: 1},
		},
		Edges: []gllbin.Edge{{V1: 0, V2: 1}},
		Faces: []gllbin.Face{{Vertices: []int32{0, 1}}}, // intentionally not encoded
	}

	src := SyntheticSource("V0 Geom", "sv0", 90.0)
	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = 4
	file.GenSystem.Label = "V0"
	file.GenSystem.Key = "sysV0"
	file.GenSystem.Type = gllbin.SystemTypeLoudspeaker
	file.Database = &gllbin.Database{
		SubVersion:        3,
		SourceDefinitions: []gllbin.SourceDefinitionItem{src},
		BoxTypes: []gllbin.BoxType{
			{Label: "V0 Box", Key: "boxV0", CaseGeometry: geom},
		},
	}

	var buf bytes.Buffer
	if err := EncodeFile(file, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	parsed, err := gllbin.Parse(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Database == nil || len(parsed.Database.BoxTypes) != 1 {
		t.Fatalf("expected 1 box, got %+v", parsed.Database)
	}
	got := parsed.Database.BoxTypes[0].CaseGeometry
	if got == nil {
		t.Fatal("expected CaseGeometry, got nil")
	}
	if len(got.Faces) != 0 {
		t.Errorf("sub_version 0 must drop faces, got %d", len(got.Faces))
	}
	if len(got.Vertices) != 2 || len(got.Edges) != 1 {
		t.Errorf("vertex/edge round-trip wrong: V=%d E=%d", len(got.Vertices), len(got.Edges))
	}
}
