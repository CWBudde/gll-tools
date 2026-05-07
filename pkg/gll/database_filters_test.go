package gll

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilterGroupParsing(t *testing.T) {
	cases := []struct {
		file        string
		wantFilters int
	}{
		{"3Way-LR.gll", 9},
		{"APS-V1_1.gll", 3},
		{"CoRay4-V1_5.gll", 3},
		{"CoRay4-Twin-V1_5.gll", 5},
		{"HOPS7-Pro V1_0.gll", 5},
		{"N-APS v1_0.gll", 2},
		{"N-RAY-V0_3 Beta.gll", 3},
		{"SCP-F-Sub Array V1_0.gll", 1},
		{"SCP-F-V1_0.gll", 2},
		{"TiRAY-V1_3.gll", 3},
		{"D12-v10.gll", 0},
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
				if tc.wantFilters == 0 {
					return
				}
				t.Fatalf("Database is nil, expected %d filter groups", tc.wantFilters)
			}

			got := len(gllFile.Database.FilterGroups)
			if got != tc.wantFilters {
				t.Errorf("FilterGroups count = %d, want %d", got, tc.wantFilters)
			}

			for i, fg := range gllFile.Database.FilterGroups {
				if fg.Key == "" {
					t.Errorf("FilterGroups[%d].Key is empty", i)
				}
				if fg.Label == "" {
					t.Errorf("FilterGroups[%d].Label is empty", i)
				}
			}
		})
	}
}

func TestFilterGroupDefinitions(t *testing.T) {
	// 3Way-LR has 9 filter groups with actual filter definitions
	f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", "3Way-LR.gll"))
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

	if len(gllFile.Database.FilterGroups) != 9 {
		t.Errorf("expected 9 FilterGroups, got %d", len(gllFile.Database.FilterGroups))
	}

	// At least one filter group should have filter definitions
	hasFilters := false
	for i, fg := range gllFile.Database.FilterGroups {
		if len(fg.Filters) > 0 {
			hasFilters = true
			for j, fd := range fg.Filters {
				if fd.Key == "" {
					t.Errorf("FilterGroups[%d].Filters[%d].Key is empty", i, j)
				}
			}
		}
	}

	if !hasFilters {
		t.Error("expected at least one FilterGroup to contain FilterDefinitions")
	}
}

func TestFilterGroupUniqueKeys(t *testing.T) {
	files := []string{"3Way-LR.gll", "HOPS7-Pro V1_0.gll", "TiRAY-V1_3.gll"}

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

			keys := make(map[string]bool)
			for i, fg := range gllFile.Database.FilterGroups {
				if fg.Key == "" {
					continue
				}
				if keys[fg.Key] {
					t.Errorf("FilterGroups[%d]: duplicate key %q", i, fg.Key)
				}
				keys[fg.Key] = true
			}
		})
	}
}

func TestIIRFilterParams(t *testing.T) {
	// CoRay4 has IIR-based filter groups
	f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", "CoRay4-V1_5.gll"))
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

	foundIIR := false
	for _, fg := range gllFile.Database.FilterGroups {
		for _, fd := range fg.Filters {
			if fd.Filter == nil {
				continue
			}
			for _, baseFilter := range fd.Filter.Filters {
				if baseFilter.Kind == FilterKindIIR && baseFilter.IIRParams != nil {
					foundIIR = true
					p := baseFilter.IIRParams
					if p.Order <= 0 || p.Order > 8 {
						t.Errorf("IIR filter order %d out of range [1,8]", p.Order)
					}
					if p.FreqCritInHz < 0 {
						t.Errorf("FreqCritInHz = %v, expected >= 0", p.FreqCritInHz)
					}
				}
			}
		}
	}

	if !foundIIR {
		t.Log("no IIR filters found (file may use LogSpectrum format)")
	}
}
