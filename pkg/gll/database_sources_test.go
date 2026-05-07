package gll

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSourceDefinitionParsing(t *testing.T) {
	cases := []struct {
		file        string
		wantSources int
	}{
		{"APS-V1_1.gll", 7},
		{"TiRAY-V1_3.gll", 4},
		{"N-APS v1_0.gll", 6},
		{"HOPS7-Pro V1_0.gll", 8},
		{"Coda-Audio G-Series-V1_2.gll", 8},
		{"3Way-LR.gll", 3},
		{"CoRay4-V1_5.gll", 3},
		{"D12-v10.gll", 1},
		{"D20-V10.gll", 1},
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
				if tc.wantSources == 0 {
					return
				}
				t.Fatalf("Database is nil, expected %d sources", tc.wantSources)
			}

			got := len(gllFile.Database.SourceDefinitions)
			if got != tc.wantSources {
				t.Errorf("SourceDefinitions count = %d, want %d", got, tc.wantSources)
			}

			for i, src := range gllFile.Database.SourceDefinitions {
				if src.Key == "" {
					t.Errorf("SourceDefinitions[%d].Key is empty", i)
				}
				if src.Definition == nil {
					t.Errorf("SourceDefinitions[%d].Definition is nil", i)
					continue
				}
				if src.Definition.Label == "" {
					t.Errorf("SourceDefinitions[%d].Definition.Label is empty", i)
				}
			}
		})
	}
}

func TestSourceDefinitionBalloon(t *testing.T) {
	// D12 is a simple 1-source file — good for precise balloon validation
	f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", "D12-v10.gll"))
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	defer f.Close()

	gllFile, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if gllFile.Database == nil || len(gllFile.Database.SourceDefinitions) == 0 {
		t.Fatal("expected at least one SourceDefinition")
	}

	src := gllFile.Database.SourceDefinitions[0]
	if src.Definition == nil {
		t.Fatal("Definition is nil")
	}

	def := src.Definition
	if def.BalloonData == nil {
		t.Fatal("BalloonData is nil — expected directivity data")
	}

	balloon := def.BalloonData

	// Standard GLL balloon grid: 5° resolution → 72 meridians × 37 parallels
	res := balloon.AngularResolution
	if res.MeridianStep != 5.0 {
		t.Errorf("MeridianStep = %v, want 5.0", res.MeridianStep)
	}
	if res.ParallelStep != 5.0 {
		t.Errorf("ParallelStep = %v, want 5.0", res.ParallelStep)
	}
	if res.MeridianCount() != 72 {
		t.Errorf("MeridianCount = %d, want 72", res.MeridianCount())
	}
	if res.ParallelCount() != 37 {
		t.Errorf("ParallelCount = %d, want 37", res.ParallelCount())
	}

	if balloon.ResponseCount <= 0 {
		t.Errorf("ResponseCount = %d, expected > 0", balloon.ResponseCount)
	}
}

func TestSourceDefinitionFrequencyRange(t *testing.T) {
	// All sources should have valid frequency ranges
	files := []string{"D12-v10.gll", "APS-V1_1.gll", "TiRAY-V1_3.gll"}

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

			if gllFile.Database == nil {
				t.Skip("no database")
			}

			for i, src := range gllFile.Database.SourceDefinitions {
				if src.Definition == nil {
					continue
				}
				def := src.Definition
				if def.NominalBandwidthFrom < 0 {
					t.Errorf("SourceDefinitions[%d]: NominalBandwidthFrom = %v, expected >= 0", i, def.NominalBandwidthFrom)
				}
				if def.NominalBandwidthTo < 0 {
					t.Errorf("SourceDefinitions[%d]: NominalBandwidthTo = %v, expected >= 0", i, def.NominalBandwidthTo)
				}
				if def.NominalBandwidthFrom > def.NominalBandwidthTo && def.NominalBandwidthTo != 0 {
					t.Errorf("SourceDefinitions[%d]: BandwidthFrom (%v) > BandwidthTo (%v)", i, def.NominalBandwidthFrom, def.NominalBandwidthTo)
				}
			}
		})
	}
}

func TestSourceDefinitionMultiSource(t *testing.T) {
	// HOPS7-Pro has 8 sources — good for multi-source validation
	f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", "HOPS7-Pro V1_0.gll"))
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

	if len(gllFile.Database.SourceDefinitions) != 8 {
		t.Errorf("expected 8 SourceDefinitions, got %d", len(gllFile.Database.SourceDefinitions))
	}

	keys := make(map[string]bool)
	for i, src := range gllFile.Database.SourceDefinitions {
		if src.Key == "" {
			t.Errorf("SourceDefinitions[%d].Key is empty", i)
			continue
		}
		if keys[src.Key] {
			t.Errorf("duplicate source key: %q", src.Key)
		}
		keys[src.Key] = true
	}
}
