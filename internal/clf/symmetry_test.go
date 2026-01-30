package clf

import "testing"

func TestGLLSymmetryToCLF(t *testing.T) {
	tests := []struct {
		gllSym int32
		want   string
	}{
		{SymmetryNone, "<none>"},
		{SymmetryVertical, "<vertical>"},
		{SymmetryHorizontal, "<horizontal>"},
		{SymmetryQuarter, "<full>"},
		{SymmetryAxial, "<rotational>"},
		{99, "<none>"},
	}

	for _, tt := range tests {
		got := GLLSymmetryToCLF(tt.gllSym)
		if got != tt.want {
			t.Errorf("GLLSymmetryToCLF(%d) = %q, want %q", tt.gllSym, got, tt.want)
		}
	}
}

func TestCLFAzimuthCount(t *testing.T) {
	tests := []struct {
		name   string
		gllSym int32
		step   float64
		want   int
	}{
		{"none 5deg", SymmetryNone, 5, 72},
		{"horizontal 5deg", SymmetryHorizontal, 5, 37},
		{"vertical 5deg", SymmetryVertical, 5, 37},
		{"quarter 5deg", SymmetryQuarter, 5, 19},
		{"axial", SymmetryAxial, 5, 1},
		{"none 10deg", SymmetryNone, 10, 36},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AzimuthCount(tt.gllSym, tt.step)
			if got != tt.want {
				t.Errorf("AzimuthCount(%d, %v) = %d, want %d", tt.gllSym, tt.step, got, tt.want)
			}
		})
	}
}
