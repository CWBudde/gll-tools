package xgll

import (
	"bytes"
	"os"
	"testing"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// TestEncoderSynthetic_BoxTypeFull builds a programmatic *gllbin.File rich
// enough to exercise encodeBoxType (with InputConfig + CaseGeometry + opening
// angles + source placements), encodeBoxSource, encodeBoxInputBuffer,
// encodeBoxInput (with SourceFilterLink + RatedImpedance), and the geometry
// encoder family. After encoding via EncodeFile we re-parse with gllbin.Parse
// and assert the round-trip preserves the structure.
func TestEncoderSynthetic_BoxTypeFull(t *testing.T) {
	src := SyntheticSource("Full Range", "src1", 90.0)

	box := gllbin.BoxType{
		Label: "TestBox",
		Key:   "boxKey",
		SourcePlacements: []gllbin.BoxSource{
			{
				Label:        "front",
				Key:          "bsKey",
				Position:     gllbin.Vector3D{X: 0, Y: 0, Z: 0},
				Angles:       gllbin.Vector3D{X: 0, Y: 0, Z: 0},
				SourceDefKey: "src1",
			},
		},
		InputConfig: &gllbin.BoxInputConfig{
			Label: "Bi-Amp",
			Key:   "ic1",
			Inputs: []gllbin.BoxInput{
				{
					Label: "LF",
					SourceLinks: []gllbin.SourceFilterLink{
						{SourceKey: "src1", FilterGrpKey: "fg1"},
					},
					RatedImpedance: 8.0,
				},
				{
					Label: "HF",
				},
			},
		},
		CaseGeometry: &gllbin.CaseGeometry{
			IsSymmetric:  true,
			SymmetryAxis: 0.0,
			Vertices: []gllbin.Vertex{
				{Color: 0xFFFFFF, X: 0, Y: 0, Z: 0, Label: "v0"},
				{Color: 0xFFFFFF, X: 1, Y: 0, Z: 0, Label: "v1"},
				{Color: 0xFFFFFF, X: 0, Y: 1, Z: 0, Label: "v2"},
			},
			Edges: []gllbin.Edge{
				{Color: 0, V1: 0, V2: 1, Label: "e0"},
				{Color: 0, V1: 1, V2: 2, Label: "e1"},
			},
			SubVersion: 0, // sver=0 → no faces, exercising the no-face branch
		},
		NextPivot:              &gllbin.Vector3D{X: 0, Y: 0, Z: 0.5},
		ReferencePoint:         &gllbin.Vector3D{X: 0, Y: 0, Z: 0},
		CenterOfMass:           &gllbin.Vector3D{X: 0, Y: 0, Z: 0.25},
		Weight:                 12.5,
		VerticalOpeningAngle:   60.0, // triggers subver=1 branch in encodeBoxType
		HorizontalOpeningAngle: 90.0,
	}

	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = 4
	file.GenSystem.Label = "Synthetic"
	file.GenSystem.Key = "synSys"
	file.GenSystem.Type = gllbin.SystemTypeLoudspeaker
	file.GenSystem.SubVersion = 3
	file.GenSystem.FlagsPresent = true
	file.GenSystem.AllowUserDefinedClusterSetup = true
	file.GenSystem.EnableForSubArrays = true
	file.Database = &gllbin.Database{
		SubVersion:        3,
		SourceDefinitions: []gllbin.SourceDefinitionItem{src},
		BoxTypes:          []gllbin.BoxType{box},
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
		t.Fatal("Database is nil")
	}
	if len(parsed.Database.BoxTypes) != 1 {
		t.Fatalf("BoxTypes len = %d, want 1", len(parsed.Database.BoxTypes))
	}

	got := parsed.Database.BoxTypes[0]
	if got.Label != "TestBox" || got.Key != "boxKey" {
		t.Errorf("BoxType Label/Key = %q/%q", got.Label, got.Key)
	}
	if len(got.SourcePlacements) != 1 || got.SourcePlacements[0].SourceDefKey != "src1" {
		t.Errorf("SourcePlacements = %+v", got.SourcePlacements)
	}
	if got.VerticalOpeningAngle != 60.0 {
		t.Errorf("VerticalOpeningAngle = %v, want 60", got.VerticalOpeningAngle)
	}
	if got.HorizontalOpeningAngle != 90.0 {
		t.Errorf("HorizontalOpeningAngle = %v, want 90", got.HorizontalOpeningAngle)
	}
	if got.InputConfig == nil {
		t.Fatal("InputConfig is nil after round-trip")
	}
	if got.InputConfig.Label != "Bi-Amp" || got.InputConfig.Key != "ic1" {
		t.Errorf("InputConfig Label/Key = %q/%q", got.InputConfig.Label, got.InputConfig.Key)
	}
	if len(got.InputConfig.Inputs) != 2 {
		t.Fatalf("Inputs len = %d, want 2", len(got.InputConfig.Inputs))
	}
	if got.InputConfig.Inputs[0].Label != "LF" || got.InputConfig.Inputs[1].Label != "HF" {
		t.Errorf("input labels = %q, %q", got.InputConfig.Inputs[0].Label, got.InputConfig.Inputs[1].Label)
	}
	if got.InputConfig.Inputs[0].RatedImpedance != 8.0 {
		t.Errorf("RatedImpedance = %v, want 8", got.InputConfig.Inputs[0].RatedImpedance)
	}
	if len(got.InputConfig.Inputs[0].SourceLinks) != 1 ||
		got.InputConfig.Inputs[0].SourceLinks[0].SourceKey != "src1" {
		t.Errorf("SourceLinks = %+v", got.InputConfig.Inputs[0].SourceLinks)
	}
	if got.CaseGeometry == nil {
		t.Fatal("CaseGeometry is nil after round-trip")
	}
	if !got.CaseGeometry.IsSymmetric {
		t.Error("CaseGeometry.IsSymmetric should be true")
	}
	if len(got.CaseGeometry.Vertices) != 3 {
		t.Errorf("Vertices len = %d, want 3", len(got.CaseGeometry.Vertices))
	}
	if len(got.CaseGeometry.Edges) != 2 {
		t.Errorf("Edges len = %d, want 2", len(got.CaseGeometry.Edges))
	}
	if got.Weight != 12.5 {
		t.Errorf("Weight = %v, want 12.5", got.Weight)
	}
}

// TestRoundTripSourceDefinitionsViaXGLLText verifies that SourceDefinitions
// (with full BalloonData and OnAxisSpectrum) survive a GLL → XGLL text → GLL
// round-trip. The XGLL text encoder serializes each item as metadata
// statements + a BinarySourceDefinition base64 blob; the parser inflates the
// blob back into a SourceDefinitionItem.
func TestRoundTripSourceDefinitionsViaXGLLText(t *testing.T) {
	src := SyntheticSource("Mid-High", "src1", 95.0)
	src.Definition.CompanyLabel = "ACME Audio"
	src.Definition.Description = "synthetic test source"

	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = 4
	file.GenSystem.Label = "Synthetic"
	file.GenSystem.Key = "synSys"
	file.GenSystem.Type = gllbin.SystemTypeLoudspeaker
	file.GenSystem.SubVersion = 3
	file.Database = &gllbin.Database{
		SubVersion:        3,
		SourceDefinitions: []gllbin.SourceDefinitionItem{src},
	}

	// Round 1: GLL model → XGLL document → XGLL text bytes.
	doc, err := BuildXGLLDocument(file)
	if err != nil {
		t.Fatalf("build xgll: %v", err)
	}
	var textBuf bytes.Buffer
	if err := WriteXGLL(doc, &textBuf); err != nil {
		t.Fatalf("write xgll text: %v", err)
	}

	// Round 2: parse XGLL text → document → GLL model → binary GLL bytes.
	parsedDoc, err := Parse(&textBuf)
	if err != nil {
		t.Fatalf("parse xgll text: %v", err)
	}
	roundFile, err := BuildGLLFile(parsedDoc)
	if err != nil {
		t.Fatalf("build gll file: %v", err)
	}
	if roundFile.Database == nil || len(roundFile.Database.SourceDefinitions) != 1 {
		t.Fatalf("expected 1 SourceDefinition, got db=%v", roundFile.Database)
	}

	got := roundFile.Database.SourceDefinitions[0]
	if got.Key != "src1" {
		t.Errorf("Key = %q, want src1", got.Key)
	}
	if got.Definition == nil {
		t.Fatal("Definition is nil after XGLL text round-trip")
	}
	def := got.Definition
	if def.Label != "Mid-High" {
		t.Errorf("Label = %q, want Mid-High", def.Label)
	}
	if def.CompanyLabel != "ACME Audio" {
		t.Errorf("CompanyLabel = %q, want ACME Audio", def.CompanyLabel)
	}
	if def.Description != "synthetic test source" {
		t.Errorf("Description = %q, want synthetic test source", def.Description)
	}
	if def.OnAxisLevel != 95.0 {
		t.Errorf("OnAxisLevel = %v, want 95", def.OnAxisLevel)
	}
	if def.NominalBandwidthFrom != 50.0 || def.NominalBandwidthTo != 5000.0 {
		t.Errorf("Bandwidth = %v..%v, want 50..5000", def.NominalBandwidthFrom, def.NominalBandwidthTo)
	}
	if def.DataType != gllbin.DataTypeThirdOctave {
		t.Errorf("DataType = %v, want ThirdOctave", def.DataType)
	}
	if def.BalloonData == nil {
		t.Fatal("BalloonData is nil after round-trip")
	}
	wantParallels := src.Definition.BalloonData.AngularResolution.ParallelCount()
	if len(def.BalloonData.Responses) != wantParallels {
		t.Errorf("balloon responses = %d, want %d", len(def.BalloonData.Responses), wantParallels)
	}
	if def.OnAxisSpectrum == nil {
		t.Fatal("OnAxisSpectrum is nil after round-trip")
	}
	if def.OnAxisSpectrum.Definition.PointCount != src.Definition.OnAxisSpectrum.Definition.PointCount {
		t.Errorf("OnAxisSpectrum.PointCount = %d, want %d",
			def.OnAxisSpectrum.Definition.PointCount,
			src.Definition.OnAxisSpectrum.Definition.PointCount)
	}

	// Round 3: write the round-tripped File as binary GLL and re-parse to
	// confirm the binary path also survives.
	var binBuf bytes.Buffer
	if err := EncodeFile(roundFile, &binBuf); err != nil {
		t.Fatalf("encode binary gll: %v", err)
	}
	finalFile, err := gllbin.Parse(bytes.NewReader(binBuf.Bytes()))
	if err != nil {
		t.Fatalf("parse final gll: %v", err)
	}
	if finalFile.Database == nil || len(finalFile.Database.SourceDefinitions) != 1 {
		t.Fatalf("final gll missing source defs: %v", finalFile.Database)
	}
	finalDef := finalFile.Database.SourceDefinitions[0].Definition
	if finalDef == nil || finalDef.BalloonData == nil {
		t.Fatal("final gll lost balloon data")
	}
	// gllbin.Parse uses lazy loading for balloon responses, so Responses is
	// nil here; ResponseCount is what we can assert without an extra load.
	//nolint:gosec // wantParallels is bounded by the synthetic resolution
	if finalDef.BalloonData.ResponseCount != int32(wantParallels) {
		t.Errorf("final ResponseCount = %d, want %d",
			finalDef.BalloonData.ResponseCount, wantParallels)
	}
}

// TestRoundTripFilterGroupsViaXGLLText loads a real GLL fixture with
// FilterGroups, builds an XGLL document, writes it as text, parses it back,
// and verifies the FilterGroup metadata + filter definitions survive the
// round-trip. The XGLL text emits each FilterGroup as human-readable
// metadata + a BinaryFilterGroup base64 blob; the parser inflates the blob
// via gllbin.ParseFilterGroupBytes to recover the full filter bank data.
func TestRoundTripFilterGroupsViaXGLLText(t *testing.T) {
	const fixture = "../../testdata/gll/3Way-LR.gll"

	f, err := os.Open(fixture)
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	defer f.Close()

	origFile, err := gllbin.Parse(f)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	if origFile.Database == nil || len(origFile.Database.FilterGroups) == 0 {
		t.Fatalf("fixture has no FilterGroups")
	}
	origGroups := origFile.Database.FilterGroups

	// GLL → XGLL document → XGLL text bytes
	doc, err := BuildXGLLDocument(origFile)
	if err != nil {
		t.Fatalf("build xgll: %v", err)
	}
	var textBuf bytes.Buffer
	if err := WriteXGLL(doc, &textBuf); err != nil {
		t.Fatalf("write xgll text: %v", err)
	}

	// XGLL text → document → File
	parsedDoc, err := Parse(&textBuf)
	if err != nil {
		t.Fatalf("parse xgll text: %v", err)
	}
	roundFile, err := BuildGLLFile(parsedDoc)
	if err != nil {
		t.Fatalf("build gll file: %v", err)
	}

	if roundFile.Database == nil {
		t.Fatal("round-tripped database is nil")
	}
	gotGroups := roundFile.Database.FilterGroups
	if len(gotGroups) != len(origGroups) {
		t.Fatalf("FilterGroups count = %d, want %d", len(gotGroups), len(origGroups))
	}

	for i, want := range origGroups {
		got := gotGroups[i]
		if got.Label != want.Label {
			t.Errorf("FilterGroups[%d].Label = %q, want %q", i, got.Label, want.Label)
		}
		if got.Key != want.Key {
			t.Errorf("FilterGroups[%d].Key = %q, want %q", i, got.Key, want.Key)
		}
		if got.IsOverridable != want.IsOverridable {
			t.Errorf("FilterGroups[%d].IsOverridable = %v, want %v",
				i, got.IsOverridable, want.IsOverridable)
		}
		if len(got.Filters) != len(want.Filters) {
			t.Errorf("FilterGroups[%d].Filters count = %d, want %d",
				i, len(got.Filters), len(want.Filters))
			continue
		}
		for j, wantFilter := range want.Filters {
			gotFilter := got.Filters[j]
			if gotFilter.Label != wantFilter.Label {
				t.Errorf("FilterGroups[%d].Filters[%d].Label = %q, want %q",
					i, j, gotFilter.Label, wantFilter.Label)
			}
			if gotFilter.Key != wantFilter.Key {
				t.Errorf("FilterGroups[%d].Filters[%d].Key = %q, want %q",
					i, j, gotFilter.Key, wantFilter.Key)
			}
		}
	}
}

// TestReverseSymmetryCode_AllValues covers each switch arm of
// reverseSymmetryCode.
func TestReverseSymmetryCode_AllValues(t *testing.T) {
	tests := []struct {
		name     string
		internal int32
		want     int32
	}{
		{"axial → 0", int32(gllbin.SymmetryAxial), 0},
		{"quarter → 1", int32(gllbin.SymmetryQuarter), 1},
		{"vertical → 2", int32(gllbin.SymmetryVertical), 2},
		{"horizontal → 3", int32(gllbin.SymmetryHorizontal), 3},
		{"none → 4", int32(gllbin.SymmetryNone), 4},
		{"unknown → default 4", 99, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reverseSymmetryCode(tt.internal); got != tt.want {
				t.Errorf("reverseSymmetryCode(%d) = %d, want %d", tt.internal, got, tt.want)
			}
		})
	}
}

// TestClampInt16_Bounds covers the over/under-range paths of clampInt16.
func TestClampInt16_Bounds(t *testing.T) {
	tests := []struct {
		in   float64
		want int16
	}{
		{0, 0},
		{100, 100},
		{-100, -100},
		{40000, 32767},   // overflow → MaxInt16
		{-40000, -32768}, // underflow → MinInt16
	}
	for _, tt := range tests {
		got := clampInt16(tt.in)
		if got != tt.want {
			t.Errorf("clampInt16(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// TestRoundTripLimitsWarningsViaXGLLText loads a real GLL fixture with
// Limits and Warnings, builds an XGLL document, writes it as text, parses
// it back, and verifies the metadata + raw block round-trips. Uses
// APS-V1_1.gll (L=11 W=2 C=23 F=3).
func TestRoundTripLimitsWarningsViaXGLLText(t *testing.T) {
	const fixture = "../../testdata/gll/APS-V1_1.gll"

	f, err := os.Open(fixture)
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	defer f.Close()

	origFile, err := gllbin.Parse(f)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	if origFile.Database == nil {
		t.Fatalf("fixture has no Database")
	}
	if len(origFile.Database.Limits) == 0 {
		t.Fatalf("fixture has no Limits")
	}
	if len(origFile.Database.Warnings) == 0 {
		t.Fatalf("fixture has no Warnings")
	}

	doc, err := BuildXGLLDocument(origFile)
	if err != nil {
		t.Fatalf("build xgll: %v", err)
	}
	var textBuf bytes.Buffer
	if err := WriteXGLL(doc, &textBuf); err != nil {
		t.Fatalf("write xgll text: %v", err)
	}

	parsedDoc, err := Parse(&textBuf)
	if err != nil {
		t.Fatalf("parse xgll text: %v", err)
	}
	roundFile, err := BuildGLLFile(parsedDoc)
	if err != nil {
		t.Fatalf("build gll file: %v", err)
	}

	if roundFile.Database == nil {
		t.Fatal("round-tripped database is nil")
	}

	gotLimits := roundFile.Database.Limits
	if len(gotLimits) != len(origFile.Database.Limits) {
		t.Fatalf("Limits count = %d, want %d", len(gotLimits), len(origFile.Database.Limits))
	}
	for i, want := range origFile.Database.Limits {
		got := gotLimits[i]
		if got.Frame != want.Frame || got.Type != want.Type ||
			got.BoxType != want.BoxType || got.LimitValue != want.LimitValue {
			t.Errorf("Limits[%d] = %+v, want %+v", i, got, want)
		}
	}

	gotWarnings := roundFile.Database.Warnings
	if len(gotWarnings) != len(origFile.Database.Warnings) {
		t.Fatalf("Warnings count = %d, want %d", len(gotWarnings), len(origFile.Database.Warnings))
	}
	for i, want := range origFile.Database.Warnings {
		got := gotWarnings[i]
		if got.Frame != want.Frame || got.Type != want.Type ||
			got.Text != want.Text || got.LimitValue != want.LimitValue {
			t.Errorf("Warnings[%d] = %+v, want %+v", i, got, want)
		}
	}
}

// TestRoundTripConnectorsViaXGLLText loads APS-V1_1.gll (23 connectors),
// round-trips through XGLL text, and verifies the BinaryConnector blob
// recovers the LabeledValueD angle list.
func TestRoundTripConnectorsViaXGLLText(t *testing.T) {
	const fixture = "../../testdata/gll/APS-V1_1.gll"

	f, err := os.Open(fixture)
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	defer f.Close()

	origFile, err := gllbin.Parse(f)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	if origFile.Database == nil || len(origFile.Database.Connectors) == 0 {
		t.Fatalf("fixture has no Connectors")
	}

	doc, err := BuildXGLLDocument(origFile)
	if err != nil {
		t.Fatalf("build xgll: %v", err)
	}
	var textBuf bytes.Buffer
	if err := WriteXGLL(doc, &textBuf); err != nil {
		t.Fatalf("write xgll text: %v", err)
	}
	parsedDoc, err := Parse(&textBuf)
	if err != nil {
		t.Fatalf("parse xgll text: %v", err)
	}
	roundFile, err := BuildGLLFile(parsedDoc)
	if err != nil {
		t.Fatalf("build gll file: %v", err)
	}
	gotConnectors := roundFile.Database.Connectors
	if len(gotConnectors) != len(origFile.Database.Connectors) {
		t.Fatalf("Connectors count = %d, want %d", len(gotConnectors), len(origFile.Database.Connectors))
	}
	for i, want := range origFile.Database.Connectors {
		got := gotConnectors[i]
		if got.UpperBox != want.UpperBox || got.LowerBox != want.LowerBox ||
			got.Frame != want.Frame {
			t.Errorf("Connectors[%d] keys mismatch: got %+v, want %+v", i, got, want)
		}
		if len(got.Angles) != len(want.Angles) {
			t.Errorf("Connectors[%d].Angles count = %d, want %d",
				i, len(got.Angles), len(want.Angles))
			continue
		}
		for j, wa := range want.Angles {
			ga := got.Angles[j]
			if ga.Label != wa.Label || ga.Value != wa.Value {
				t.Errorf("Connectors[%d].Angles[%d] = %+v, want %+v", i, j, ga, wa)
			}
		}
	}
}

// TestRoundTripFramesViaXGLLText loads APS-V1_1.gll (3 frames), round-trips
// through XGLL text, and verifies BinaryFrame blob inflation recovers
// CaseGeometry/PinPoints/etc.
func TestRoundTripFramesViaXGLLText(t *testing.T) {
	const fixture = "../../testdata/gll/APS-V1_1.gll"

	f, err := os.Open(fixture)
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	defer f.Close()

	origFile, err := gllbin.Parse(f)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	if origFile.Database == nil || len(origFile.Database.Frames) == 0 {
		t.Fatalf("fixture has no Frames")
	}

	doc, err := BuildXGLLDocument(origFile)
	if err != nil {
		t.Fatalf("build xgll: %v", err)
	}
	var textBuf bytes.Buffer
	if err := WriteXGLL(doc, &textBuf); err != nil {
		t.Fatalf("write xgll text: %v", err)
	}
	parsedDoc, err := Parse(&textBuf)
	if err != nil {
		t.Fatalf("parse xgll text: %v", err)
	}
	roundFile, err := BuildGLLFile(parsedDoc)
	if err != nil {
		t.Fatalf("build gll file: %v", err)
	}

	gotFrames := roundFile.Database.Frames
	if len(gotFrames) != len(origFile.Database.Frames) {
		t.Fatalf("Frames count = %d, want %d", len(gotFrames), len(origFile.Database.Frames))
	}
	for i, want := range origFile.Database.Frames {
		got := gotFrames[i]
		if got.Label != want.Label || got.Key != want.Key {
			t.Errorf("Frames[%d] label/key = (%q,%q), want (%q,%q)",
				i, got.Label, got.Key, want.Label, want.Key)
		}
		if got.TypeFlown != want.TypeFlown {
			t.Errorf("Frames[%d].TypeFlown = %v, want %v", i, got.TypeFlown, want.TypeFlown)
		}
		if got.Weight != want.Weight {
			t.Errorf("Frames[%d].Weight = %v, want %v", i, got.Weight, want.Weight)
		}
		if len(got.PinPoints) != len(want.PinPoints) {
			t.Errorf("Frames[%d].PinPoints count = %d, want %d",
				i, len(got.PinPoints), len(want.PinPoints))
		}
	}
}

// TestEncodeBlock_Roundtrip exercises the encodeBlock helper. encodeBlock
// prepends [int32 totalSize][int16 vcheck=0][int16 subVersion] before content.
func TestEncodeBlock_Roundtrip(t *testing.T) {
	content := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	block := encodeBlock(7, content)
	// totalSize header (4) + vcheck(2) + subVer(2) + content(4) = 12 bytes.
	if len(block) != 12 {
		t.Errorf("block len = %d, want 12", len(block))
	}
	// Bytes 4..6 are vcheck (int16 little-endian, value 0).
	if block[4] != 0 || block[5] != 0 {
		t.Errorf("vcheck bytes = %x %x, want 00 00", block[4], block[5])
	}
	// Bytes 6..8 are subVersion (int16 little-endian, value 7).
	if block[6] != 7 || block[7] != 0 {
		t.Errorf("subVer bytes = %x %x, want 07 00", block[6], block[7])
	}
}
