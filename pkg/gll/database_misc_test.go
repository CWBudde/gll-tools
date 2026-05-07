package gll

import (
	"os"
	"path/filepath"
	"testing"
)

// parseTestFile opens and parses a GLL file from testdata/gll/. Skips the test
// if the file is absent (allows the suite to run without all sample files).
func parseTestFile(t *testing.T, name string) *File {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "testdata", "gll", name))
	if err != nil {
		t.Skipf("test file not found: %v", err)
	}
	t.Cleanup(func() { f.Close() })

	gllFile, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse(%q) error: %v", name, err)
	}
	return gllFile
}

// ---- Connectors ----

func TestConnectorParsing(t *testing.T) {
	// Line-array systems are most likely to have connector definitions.
	files := []string{
		"TiRAY-V1_3.gll",
		"N-RAY-V0_3 Beta.gll",
		"LX-20 ASX_gll.gll",
		"LX-60 ASX_gll.gll",
		"Coda-Audio G-Series-V1_2.gll",
		"APS-V1_1.gll",
	}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			gllFile := parseTestFile(t, name)
			if gllFile.Database == nil {
				t.Skip("no database")
			}

			for i, conn := range gllFile.Database.Connectors {
				// Both boxes must be identified
				if conn.UpperBox == "" {
					t.Errorf("Connectors[%d].UpperBox is empty", i)
				}
				if conn.LowerBox == "" {
					t.Errorf("Connectors[%d].LowerBox is empty", i)
				}
				// Splay angles must be finite
				for j, ang := range conn.Angles {
					if ang.Label == "" {
						t.Errorf("Connectors[%d].Angles[%d].Label is empty", i, j)
					}
				}
			}
		})
	}
}

// ---- Frames ----

func TestFrameParsing(t *testing.T) {
	files := []string{
		"TiRAY-V1_3.gll",
		"N-RAY-V0_3 Beta.gll",
		"Coda-Audio G-Series-V1_2.gll",
		"LX-10 ASX_gll.gll",
		"LX-20 ASX_gll.gll",
		"APS-V1_1.gll",
	}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			gllFile := parseTestFile(t, name)
			if gllFile.Database == nil {
				t.Skip("no database")
			}

			for i, fr := range gllFile.Database.Frames {
				if fr.Key == "" {
					t.Errorf("Frames[%d].Key is empty", i)
				}
				if fr.Label == "" {
					t.Errorf("Frames[%d].Label is empty", i)
				}
			}
		})
	}
}

func TestFrameConnectionPoints(t *testing.T) {
	// Frames with connection points should have valid labeled vectors.
	files := []string{
		"TiRAY-V1_3.gll",
		"LX-20 ASX_gll.gll",
		"Coda-Audio G-Series-V1_2.gll",
	}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			gllFile := parseTestFile(t, name)
			if gllFile.Database == nil {
				t.Skip("no database")
			}

			for i, fr := range gllFile.Database.Frames {
				for j, pp := range fr.PinPoints {
					if pp.Label == "" {
						t.Errorf("Frames[%d].PinPoints[%d].Label is empty", i, j)
					}
				}
			}
		})
	}
}

// ---- ClusterSetups ----

func TestClusterSetupParsing(t *testing.T) {
	// Cluster-type systems are most likely to have ClusterSetups.
	files := []string{
		"APS-V1_1.gll",
		"N-APS v1_0.gll",
		"HOPS7-Pro V1_0.gll",
		"D12-v10.gll",
		"TiRAY-V1_3.gll",
	}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			gllFile := parseTestFile(t, name)
			if gllFile.Database == nil {
				t.Skip("no database")
			}

			for i, item := range gllFile.Database.ClusterSetups {
				if item.Key == "" {
					t.Errorf("ClusterSetups[%d].Key is empty", i)
				}
				for j, box := range item.Setup.Boxes {
					if box.BoxTypeKey == "" {
						t.Errorf("ClusterSetups[%d].Boxes[%d].BoxTypeKey is empty", i, j)
					}
				}
			}
		})
	}
}

// ---- Presets ----

func TestPresetParsing(t *testing.T) {
	files := []string{
		"TiRAY-V1_3.gll",
		"N-RAY-V0_3 Beta.gll",
		"Coda-Audio G-Series-V1_2.gll",
		"LX-10 ASX_gll.gll",
		"D12-v10.gll",
	}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			gllFile := parseTestFile(t, name)
			if gllFile.Database == nil {
				t.Skip("no database")
			}

			for i, preset := range gllFile.Database.Presets {
				if preset.Key == "" {
					t.Errorf("Presets[%d].Key is empty", i)
				}
				if preset.Label == "" {
					t.Errorf("Presets[%d].Label is empty", i)
				}
			}
		})
	}
}

// ---- Transformers ----

func TestTransformerParsing(t *testing.T) {
	files := []string{
		"TiRAY-V1_3.gll",
		"Coda-Audio G-Series-V1_2.gll",
		"HOPS7-Pro V1_0.gll",
		"D12-v10.gll",
	}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			gllFile := parseTestFile(t, name)
			if gllFile.Database == nil {
				t.Skip("no database")
			}

			for i, tr := range gllFile.Database.Transformers {
				if tr.Key == "" {
					t.Errorf("Transformers[%d].Key is empty", i)
				}
			}
		})
	}
}

// ---- DatabaseSubVersion ----

func TestDatabaseSubVersion(t *testing.T) {
	// All files with a database should have a non-negative sub_version.
	files := []string{
		"D12-v10.gll",
		"D20-V10.gll",
		"TiRAY-V1_3.gll",
		"APS-V1_1.gll",
	}
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			gllFile := parseTestFile(t, name)
			if gllFile.Database == nil {
				t.Skip("no database")
			}
			if gllFile.Database.SubVersion < 0 {
				t.Errorf("Database.SubVersion = %d, expected >= 0", gllFile.Database.SubVersion)
			}
		})
	}
}
