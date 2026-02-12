package xgll

import (
	"bytes"
	"testing"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// TestInputConfigXGLLOutput verifies that BoxType.InputConfig is correctly
// converted to XGLL statements (GLL → XGLL decode).
func TestInputConfigXGLLOutput(t *testing.T) {
	tests := []struct {
		name     string
		box      gllbin.BoxType
		wantKeys []string // keywords we expect to see in output
	}{
		{
			name: "nil input config produces no statements",
			box: gllbin.BoxType{
				Label:       "Test Box",
				Key:         "bxTest",
				InputConfig: nil,
			},
			wantKeys: nil, // no input config statements
		},
		{
			name: "basic input config with one input",
			box: gllbin.BoxType{
				Label: "Test Box",
				Key:   "bxTest",
				InputConfig: &gllbin.BoxInputConfig{
					Label: "Passive",
					Key:   "icPassive",
					Inputs: []gllbin.BoxInput{
						{
							Label:          "SpeakonIn",
							RatedImpedance: 8.0,
							SourceLinks: []gllbin.SourceFilterLink{
								{SourceKey: "srcLF", FilterGrpKey: "FG_LowPass"},
							},
						},
					},
				},
			},
			wantKeys: []string{
				"Input Configurations",
				"Input Configuration",
				"Inputs",
				"Input",
				"Rated Impedance",
				"Links",
				"Link",
			},
		},
		{
			name: "input config with multiple inputs and links",
			box: gllbin.BoxType{
				Label: "Two-Way",
				Key:   "bxTwoWay",
				InputConfig: &gllbin.BoxInputConfig{
					Label: "BiAmp",
					Key:   "icBiAmp",
					Inputs: []gllbin.BoxInput{
						{
							Label:          "LF Input",
							RatedImpedance: 4.0,
							SourceLinks: []gllbin.SourceFilterLink{
								{SourceKey: "srcLF", FilterGrpKey: "FG_LFDirect"},
							},
						},
						{
							Label:          "HF Input",
							RatedImpedance: 8.0,
							SourceLinks: []gllbin.SourceFilterLink{
								{SourceKey: "srcHF", FilterGrpKey: "FG_HFDirect"},
							},
						},
					},
				},
			},
			wantKeys: []string{
				"Input Configurations",
				"Input Configuration",
				"Inputs",
				"Input",
			},
		},
		{
			name: "input without rated impedance (default 0)",
			box: gllbin.BoxType{
				Label: "Test Box",
				Key:   "bxTest",
				InputConfig: &gllbin.BoxInputConfig{
					Label: "Simple",
					Key:   "icSimple",
					Inputs: []gllbin.BoxInput{
						{
							Label:          "Main",
							RatedImpedance: 0, // default, should not produce "Rated Impedance" statement
							SourceLinks: []gllbin.SourceFilterLink{
								{SourceKey: "src1", FilterGrpKey: "fg1"},
							},
						},
					},
				},
			},
			wantKeys: []string{
				"Input Configurations",
				"Input Configuration",
				"Inputs",
				"Input",
				"Links",
				"Link",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build statements from BoxType
			statements := buildInputConfigStatements(tt.box.InputConfig)

			// Check expected keywords are present
			keywordSet := make(map[string]bool)
			for _, stmt := range statements {
				keywordSet[stmt.Keyword] = true
			}

			for _, want := range tt.wantKeys {
				if !keywordSet[want] {
					t.Errorf("missing expected keyword %q in output", want)
				}
			}

			// If no InputConfig, should produce no statements
			if tt.box.InputConfig == nil && len(statements) > 0 {
				t.Errorf("nil InputConfig should produce no statements, got %d", len(statements))
			}
		})
	}
}

// TestInputConfigXGLLParsing verifies that XGLL InputConfig statements are
// correctly parsed into BoxType.InputConfig (XGLL → GLL convert).
func TestInputConfigXGLLParsing(t *testing.T) {
	tests := []struct {
		name       string
		xgll       string
		wantLabel  string
		wantKey    string
		wantInputs int
		wantLinks  int
	}{
		{
			name: "basic input config",
			xgll: `"GLL"
"Format", "3D"
"FormatVersion", 1
"System", "Test", "sysTest", "LS"
"Layout"
"Data"
"BoxTypes", 1
"BoxType", "Test Box", "bxTest"
"Input Configurations", 1
"Input Configuration", "Passive", "icPassive"
"Inputs", 1
"Input", "SpeakonIn"
"Rated Impedance", 8
"Links", 2
"Link", "srcLF", "FG_LowPass"
"Link", "srcHF", "FG_HighPass"
`,
			wantLabel:  "Passive",
			wantKey:    "icPassive",
			wantInputs: 1,
			wantLinks:  2,
		},
		{
			name: "multiple inputs",
			xgll: `"GLL"
"Format", "3D"
"FormatVersion", 1
"System", "Test", "sysTest", "LS"
"Layout"
"Data"
"BoxTypes", 1
"BoxType", "BiAmp Box", "bxBiAmp"
"Input Configurations", 1
"Input Configuration", "BiAmp", "icBiAmp"
"Inputs", 2
"Input", "LF Input"
"Rated Impedance", 4
"Links", 1
"Link", "srcLF", "FG_LF"
"Input", "HF Input"
"Rated Impedance", 8
"Links", 1
"Link", "srcHF", "FG_HF"
`,
			wantLabel:  "BiAmp",
			wantKey:    "icBiAmp",
			wantInputs: 2,
			wantLinks:  1, // per input
		},
		{
			name: "no input config",
			xgll: `"GLL"
"Format", "3D"
"FormatVersion", 1
"System", "Test", "sysTest", "LS"
"Layout"
"Data"
"BoxTypes", 1
"BoxType", "Simple Box", "bxSimple"
"Sources", 1
"Source", "Main", "src1", 0, 0, 0, 0, 0, 0, "SD_Main"
`,
			wantLabel:  "",
			wantKey:    "",
			wantInputs: 0,
			wantLinks:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse XGLL document
			doc, err := Parse(bytes.NewReader([]byte(tt.xgll)))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			// Convert to GLL
			file, err := BuildGLLFile(doc)
			if err != nil {
				t.Fatalf("build gll failed: %v", err)
			}

			// Check BoxTypes
			if file.Database == nil || len(file.Database.BoxTypes) == 0 {
				if tt.wantInputs > 0 {
					t.Fatal("expected BoxTypes but got none")
				}
				return
			}

			box := file.Database.BoxTypes[0]

			// Check InputConfig presence
			if tt.wantLabel == "" {
				if box.InputConfig != nil {
					t.Errorf("expected nil InputConfig, got %+v", box.InputConfig)
				}
				return
			}

			if box.InputConfig == nil {
				t.Fatal("expected InputConfig but got nil")
			}

			// Verify label and key
			if box.InputConfig.Label != tt.wantLabel {
				t.Errorf("label: got %q, want %q", box.InputConfig.Label, tt.wantLabel)
			}
			if box.InputConfig.Key != tt.wantKey {
				t.Errorf("key: got %q, want %q", box.InputConfig.Key, tt.wantKey)
			}

			// Verify input count
			if len(box.InputConfig.Inputs) != tt.wantInputs {
				t.Errorf("inputs: got %d, want %d", len(box.InputConfig.Inputs), tt.wantInputs)
			}

			// Verify first input has expected links
			if tt.wantInputs > 0 && len(box.InputConfig.Inputs) > 0 {
				if len(box.InputConfig.Inputs[0].SourceLinks) != tt.wantLinks {
					t.Errorf("links: got %d, want %d",
						len(box.InputConfig.Inputs[0].SourceLinks), tt.wantLinks)
				}
			}
		})
	}
}

// TestInputConfigRoundTrip verifies InputConfig survives XGLL → GLL → XGLL.
func TestInputConfigRoundTrip(t *testing.T) {
	// Original InputConfig
	original := &gllbin.BoxInputConfig{
		Label: "Passive Config",
		Key:   "icPassive",
		Inputs: []gllbin.BoxInput{
			{
				Label:          "Main Input",
				RatedImpedance: 8.0,
				SourceLinks: []gllbin.SourceFilterLink{
					{SourceKey: "srcLF", FilterGrpKey: "FG_LowPass"},
					{SourceKey: "srcHF", FilterGrpKey: "FG_HighPass"},
				},
			},
		},
	}

	// Create minimal GLL file with BoxType containing InputConfig
	file := &gllbin.File{
		Header: gllbin.Header{
			Magic:         "EGLL",
			FormatID:      "EASE_GLL",
			FormatVersion: 4,
		},
		GenSystem: gllbin.GenSystem{
			Label: "Test System",
			Key:   "sysTest",
			Type:  gllbin.SystemTypeLoudspeaker,
		},
		Database: &gllbin.Database{
			BoxTypes: []gllbin.BoxType{
				{
					Label:       "Test Box",
					Key:         "bxTest",
					InputConfig: original,
				},
			},
		},
	}

	// Convert GLL → XGLL
	doc, err := BuildXGLLDocument(file)
	if err != nil {
		t.Fatalf("build xgll: %v", err)
	}

	// Write XGLL to buffer for debugging
	var xgllBuf bytes.Buffer
	if err := WriteXGLL(doc, &xgllBuf); err != nil {
		t.Fatalf("write xgll: %v", err)
	}
	t.Logf("XGLL output:\n%s", xgllBuf.String())

	// Convert XGLL → GLL
	roundFile, err := BuildGLLFile(doc)
	if err != nil {
		t.Fatalf("build gll from xgll: %v", err)
	}

	// Verify InputConfig survived
	if roundFile.Database == nil || len(roundFile.Database.BoxTypes) == 0 {
		t.Fatal("expected BoxTypes after round-trip")
	}

	box := roundFile.Database.BoxTypes[0]
	if box.InputConfig == nil {
		t.Fatal("InputConfig lost during round-trip")
	}

	// Compare fields
	if box.InputConfig.Label != original.Label {
		t.Errorf("label: got %q, want %q", box.InputConfig.Label, original.Label)
	}
	if box.InputConfig.Key != original.Key {
		t.Errorf("key: got %q, want %q", box.InputConfig.Key, original.Key)
	}
	if len(box.InputConfig.Inputs) != len(original.Inputs) {
		t.Errorf("inputs: got %d, want %d",
			len(box.InputConfig.Inputs), len(original.Inputs))
	}

	if len(box.InputConfig.Inputs) > 0 {
		gotInput := box.InputConfig.Inputs[0]
		wantInput := original.Inputs[0]

		if gotInput.Label != wantInput.Label {
			t.Errorf("input label: got %q, want %q", gotInput.Label, wantInput.Label)
		}
		if gotInput.RatedImpedance != wantInput.RatedImpedance {
			t.Errorf("rated impedance: got %f, want %f",
				gotInput.RatedImpedance, wantInput.RatedImpedance)
		}
		if len(gotInput.SourceLinks) != len(wantInput.SourceLinks) {
			t.Errorf("links: got %d, want %d",
				len(gotInput.SourceLinks), len(wantInput.SourceLinks))
		}

		for i, wantLink := range wantInput.SourceLinks {
			if i >= len(gotInput.SourceLinks) {
				break
			}
			gotLink := gotInput.SourceLinks[i]
			if gotLink.SourceKey != wantLink.SourceKey {
				t.Errorf("link[%d] source: got %q, want %q",
					i, gotLink.SourceKey, wantLink.SourceKey)
			}
			if gotLink.FilterGrpKey != wantLink.FilterGrpKey {
				t.Errorf("link[%d] filter: got %q, want %q",
					i, gotLink.FilterGrpKey, wantLink.FilterGrpKey)
			}
		}
	}
}
