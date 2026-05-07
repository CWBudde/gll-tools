package viz

import (
	"math"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

func TestComputePolarSlicesNilDef(t *testing.T) {
	if _, err := ComputePolarSlices(nil, 0, 10, false); err == nil {
		t.Error("expected error for nil def")
	}
}

func TestComputePolarSlicesNoResponses(t *testing.T) {
	def := &gll.SourceDefinition{
		BalloonData: &gll.BalloonData{
			AngularResolution: gll.ResolutionDescriptor{MeridianStep: 5, ParallelStep: 5},
		},
	}
	if _, err := ComputePolarSlices(def, 0, 10, false); err == nil {
		t.Error("expected error for unloaded responses")
	}
}

func TestComputePolarSlicesSuccess(t *testing.T) {
	// Use 45° steps → small balloon (8 meridians × 5 parallels = 40 responses).
	def := makeTestBalloon(45, 45, 90.0)

	slices, err := ComputePolarSlices(def, 0, 30, false)
	if err != nil {
		t.Fatalf("ComputePolarSlices error: %v", err)
	}
	if len(slices.AnglesDeg) == 0 {
		t.Error("expected non-empty angles")
	}
	if len(slices.HorizontalLevel) != len(slices.AnglesDeg) {
		t.Errorf("HorizontalLevel length %d != AnglesDeg length %d",
			len(slices.HorizontalLevel), len(slices.AnglesDeg))
	}
	if len(slices.VerticalLevel) != len(slices.AnglesDeg) {
		t.Errorf("VerticalLevel length %d != AnglesDeg length %d",
			len(slices.VerticalLevel), len(slices.AnglesDeg))
	}
	if slices.StepDeg != 30 {
		t.Errorf("StepDeg = %v, want 30", slices.StepDeg)
	}
}

func TestComputePolarSlicesDefaultStep(t *testing.T) {
	def := makeTestBalloon(45, 45, 90.0)
	slices, err := ComputePolarSlices(def, 0, 0, false)
	if err != nil {
		t.Fatalf("ComputePolarSlices error: %v", err)
	}
	if slices.StepDeg != 10 {
		t.Errorf("default StepDeg = %v, want 10", slices.StepDeg)
	}
}

func TestMeridianCountForSymmetry(t *testing.T) {
	cases := []struct {
		name     string
		step     float64
		symmetry int32
		full     int
		want     int
	}{
		{"axial", 5, int32(gll.SymmetryAxial), 72, 1},
		{"quarter step5", 5, int32(gll.SymmetryQuarter), 72, 19},
		{"vertical step5", 5, int32(gll.SymmetryVertical), 72, 37},
		{"horizontal step5", 5, int32(gll.SymmetryHorizontal), 72, 37},
		{"none returns full", 5, int32(gll.SymmetryNone), 72, 72},
		{"step zero", 0, int32(gll.SymmetryAxial), 72, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := meridianCountForSymmetry(tc.step, tc.symmetry, tc.full)
			if got != tc.want {
				t.Errorf("meridianCountForSymmetry(%v, %v, %v) = %d, want %d",
					tc.step, tc.symmetry, tc.full, got, tc.want)
			}
		})
	}
}

func TestParallelCountForHalf(t *testing.T) {
	cases := []struct {
		name          string
		step          float64
		frontHalfOnly bool
		full          int
		want          int
	}{
		{"step zero", 0, false, 37, 1},
		{"front half step5", 5, true, 37, 19},
		{"full returns provided", 5, false, 37, 37},
		{"full computed when zero", 5, false, 0, 37},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parallelCountForHalf(tc.step, tc.frontHalfOnly, tc.full)
			if got != tc.want {
				t.Errorf("parallelCountForHalf(%v, %v, %v) = %d, want %d",
					tc.step, tc.frontHalfOnly, tc.full, got, tc.want)
			}
		})
	}
}

func TestBuildPolarAngles(t *testing.T) {
	angles := buildPolarAngles(90)
	// Expected: [0, -90, -180, 90]
	if len(angles) == 0 {
		t.Fatal("expected non-empty angles")
	}
	if angles[0] != 0 {
		t.Errorf("first angle = %v, want 0", angles[0])
	}
	// All should be in [-180, 180]
	for _, a := range angles {
		if a < -180-1e-9 || a > 180+1e-9 {
			t.Errorf("angle %v out of [-180, 180]", a)
		}
	}
}

func TestSameSpectrumGrid(t *testing.T) {
	a := gll.LogSpectrumDefinition{BandsPerOctave: 6, StartFreq: 100, PointCount: 48}
	b := gll.LogSpectrumDefinition{BandsPerOctave: 6, StartFreq: 100, PointCount: 48}
	c := gll.LogSpectrumDefinition{BandsPerOctave: 3, StartFreq: 100, PointCount: 48}

	if !sameSpectrumGrid(a, b) {
		t.Error("expected matching grids to be reported as same")
	}
	if sameSpectrumGrid(a, c) {
		t.Error("expected differing BandsPerOctave to be reported as different")
	}
}

func TestBuildBalloonGridNil(t *testing.T) {
	if g := buildBalloonGrid(nil); g != nil {
		t.Error("expected nil grid for nil input")
	}
}

func TestBuildBalloonGridInvalidResolution(t *testing.T) {
	bd := &gll.BalloonData{
		AngularResolution: gll.ResolutionDescriptor{MeridianStep: 0, ParallelStep: 0},
	}
	if g := buildBalloonGrid(bd); g != nil {
		t.Error("expected nil grid for invalid resolution")
	}
}

func TestResponseAtAnglesNil(t *testing.T) {
	if r := ResponseAtAngles(nil, 0, 0); r != nil {
		t.Error("expected nil for nil balloon data")
	}
}

func TestComputePolarSlicesFrequency(t *testing.T) {
	def := makeTestBalloon(45, 45, 90.0)
	slices, err := ComputePolarSlices(def, 1, 30, false)
	if err != nil {
		t.Fatalf("ComputePolarSlices error: %v", err)
	}
	// freq index 1 should yield a positive frequency from spectrum definition.
	if !math.IsNaN(slices.FrequencyHz) && slices.FrequencyHz <= 0 {
		t.Errorf("FrequencyHz = %v, expected > 0", slices.FrequencyHz)
	}
}
