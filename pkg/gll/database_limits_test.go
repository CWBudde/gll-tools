package gll

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLimitParsing(t *testing.T) {
	// Files that are known to parse without errors, regardless of limit count
	files := []string{
		"D12-v10.gll",
		"D20-V10.gll",
		"TiRAY-V1_3.gll",
		"N-RAY-V0_3 Beta.gll",
		"Coda-Audio G-Series-V1_2.gll",
		"LX-10 ASX_gll.gll",
		"HOPS7-Pro V1_0.gll",
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

			if gllFile.Database == nil {
				t.Skip("no database")
			}

			// Limits may be empty — we just verify each parsed limit has valid fields
			for i, lim := range gllFile.Database.Limits {
				if lim.LimitValue < 0 {
					t.Errorf("Limits[%d].LimitValue = %v, expected >= 0", i, lim.LimitValue)
				}
				// LimitType must be a known enum value
				switch lim.Type {
				case LimitTypeMaxCount, LimitTypeMaxCountType, LimitTypeMaxWeightKg,
					LimitTypeMaxTiltAngle, LimitTypeMinTiltAngle, LimitTypeMinCount:
					// valid
				default:
					t.Errorf("Limits[%d].Type = %d, unexpected LimitType", i, lim.Type)
				}
			}
		})
	}
}

func TestWarningParsing(t *testing.T) {
	files := []string{
		"D12-v10.gll",
		"TiRAY-V1_3.gll",
		"Coda-Audio G-Series-V1_2.gll",
		"LX-20 ASX_gll.gll",
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

			if gllFile.Database == nil {
				t.Skip("no database")
			}

			for i, w := range gllFile.Database.Warnings {
				if w.LimitValue < 0 {
					t.Errorf("Warnings[%d].LimitValue = %v, expected >= 0", i, w.LimitValue)
				}
				switch w.Type {
				case WarningTypeMaxCount, WarningTypeMinCount, WarningTypeMaxWeightKg,
					WarningTypeMaxTiltAngle, WarningTypeMinTiltAngle:
					// valid
				default:
					t.Errorf("Warnings[%d].Type = %d, unexpected WarningType", i, w.Type)
				}
			}
		})
	}
}

func TestLimitTypeStringCoverage(t *testing.T) {
	cases := []struct {
		t    LimitType
		want string
	}{
		{LimitTypeMaxCount, "MaxCount"},
		{LimitTypeMaxCountType, "MaxCountType"},
		{LimitTypeMaxWeightKg, "MaxWeightKg"},
		{LimitTypeMaxTiltAngle, "MaxTiltAngle"},
		{LimitTypeMinTiltAngle, "MinTiltAngle"},
		{LimitTypeMinCount, "MinCount"},
	}
	for _, tc := range cases {
		if got := tc.t.String(); got != tc.want {
			t.Errorf("LimitType(%d).String() = %q, want %q", tc.t, got, tc.want)
		}
	}
}

func TestWarningTypeStringCoverage(t *testing.T) {
	cases := []struct {
		t    WarningType
		want string
	}{
		{WarningTypeMaxCount, "MaxCount"},
		{WarningTypeMinCount, "MinCount"},
		{WarningTypeMaxWeightKg, "MaxWeightKg"},
		{WarningTypeMaxTiltAngle, "MaxTiltAngle"},
		{WarningTypeMinTiltAngle, "MinTiltAngle"},
	}
	for _, tc := range cases {
		if got := tc.t.String(); got != tc.want {
			t.Errorf("WarningType(%d).String() = %q, want %q", tc.t, got, tc.want)
		}
	}
}
