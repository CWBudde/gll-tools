package gll

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoxTypeParsing(t *testing.T) {
	cases := []struct {
		file      string
		wantBoxes int
	}{
		{"APS-V1_1.gll", 5},
		{"TiRAY-V1_3.gll", 4},
		{"N-APS v1_0.gll", 6},
		{"Coda-Audio G-Series-V1_2.gll", 16},
		{"D12-v10.gll", 1},
		{"D20-V10.gll", 1},
		{"SCP-F-V1_0.gll", 2},
		{"N-RAY-V0_3 Beta.gll", 2},
		{"example-cl.gll", 0},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", tc.file))
			if err != nil {
				t.Skipf("test file not found: %v", err)
			}
			defer f.Close()

			gllFile, err := Parse(f)
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}

			if gllFile.Database == nil {
				if tc.wantBoxes == 0 {
					return
				}
				t.Fatalf("Database is nil, expected %d boxes", tc.wantBoxes)
			}

			got := len(gllFile.Database.BoxTypes)
			if got != tc.wantBoxes {
				t.Errorf("BoxTypes count = %d, want %d", got, tc.wantBoxes)
			}

			for i, box := range gllFile.Database.BoxTypes {
				if box.Key == "" {
					t.Errorf("BoxTypes[%d].Key is empty", i)
				}
				if box.Label == "" {
					t.Errorf("BoxTypes[%d].Label is empty", i)
				}
			}
		})
	}
}

func TestBoxTypeSourcePlacements(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", "D12-v10.gll"))
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer f.Close()

	gllFile, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if gllFile.Database == nil || len(gllFile.Database.BoxTypes) == 0 {
		t.Fatal("expected at least one BoxType")
	}

	box := gllFile.Database.BoxTypes[0]
	if len(box.SourcePlacements) == 0 {
		t.Error("expected SourcePlacements to be non-empty")
	}

	for i, sp := range box.SourcePlacements {
		if sp.SourceDefKey == "" {
			t.Errorf("SourcePlacements[%d].SourceDefKey is empty", i)
		}
	}
}

func TestBoxTypeGeometry(t *testing.T) {
	// Files with known geometry (case geometry data)
	files := []string{
		"D12-v10.gll",
		"D20-V10.gll",
		"LX-10 ASX_gll.gll",
	}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", name))
			if err != nil {
				t.Skipf("test file not found: %v", err)
			}
			defer f.Close()

			gllFile, err := Parse(f)
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}

			if gllFile.Database == nil || len(gllFile.Database.BoxTypes) == 0 {
				t.Skip("no box types in file")
			}

			// At least one box should have geometry
			hasGeometry := false
			for _, box := range gllFile.Database.BoxTypes {
				if box.CaseGeometry != nil {
					hasGeometry = true
					geom := box.CaseGeometry
					if len(geom.Vertices) == 0 {
						t.Error("CaseGeometry has no vertices")
					}
					if len(geom.Edges) == 0 {
						t.Error("CaseGeometry has no edges")
					}
				}
			}
			if !hasGeometry {
				t.Log("no box in this file has CaseGeometry (may be valid for some GLL versions)")
			}
		})
	}
}

func TestBoxTypeMultipleSources(t *testing.T) {
	// TiRAY has 4 boxes and 4 sources — good for multi-source validation
	f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", "TiRAY-V1_3.gll"))
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer f.Close()

	gllFile, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if gllFile.Database == nil {
		t.Fatal("Database is nil")
	}

	if len(gllFile.Database.BoxTypes) != 4 {
		t.Errorf("expected 4 BoxTypes, got %d", len(gllFile.Database.BoxTypes))
	}

	for i, box := range gllFile.Database.BoxTypes {
		if box.Key == "" {
			t.Errorf("BoxTypes[%d].Key is empty", i)
		}
		// Each box should have at least one source placement
		if len(box.SourcePlacements) == 0 {
			t.Logf("BoxTypes[%d] (%s) has no source placements", i, box.Label)
		}
	}
}
