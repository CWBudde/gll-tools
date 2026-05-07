package viz

import (
	"math"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// makeTestBalloon constructs a minimal SourceDefinition with synthetic
// balloon data covering meridianCount × parallelCount points at the given step.
func makeTestBalloon(meridianStep, parallelStep float64, levelDB float64) *gll.SourceDefinition {
	mCount := int(math.Round(360.0 / meridianStep))
	pCount := int(math.Round(180.0/parallelStep)) + 1

	levels := []float64{levelDB, levelDB - 3, levelDB - 6}
	specDef := gll.LogSpectrumDefinition{
		StartFreq:      100,
		BandsPerOctave: 1,
		PointCount:     int32(len(levels)),
	}

	responses := make([]gll.TransferFunction, mCount*pCount)
	for i := range responses {
		responses[i].Definition = specDef
		responses[i].Level = append([]float64{}, levels...)
	}

	return &gll.SourceDefinition{
		BalloonData: &gll.BalloonData{
			AngularResolution: gll.ResolutionDescriptor{
				MeridianStep: meridianStep,
				ParallelStep: parallelStep,
			},
			Responses:     responses,
			ResponseCount: int32(len(responses)),
		},
	}
}

func TestBuildBalloonMeshNilDef(t *testing.T) {
	if _, err := BuildBalloonMesh(nil, 0, 40, 1, false); err == nil {
		t.Error("expected error for nil def")
	}
}

func TestBuildBalloonMeshNoResponses(t *testing.T) {
	def := &gll.SourceDefinition{
		BalloonData: &gll.BalloonData{
			AngularResolution: gll.ResolutionDescriptor{MeridianStep: 5, ParallelStep: 5},
		},
	}
	if _, err := BuildBalloonMesh(def, 0, 40, 1, false); err == nil {
		t.Error("expected error when responses are not loaded")
	}
}

func TestBuildBalloonMeshInvalidResolution(t *testing.T) {
	def := &gll.SourceDefinition{
		BalloonData: &gll.BalloonData{
			AngularResolution: gll.ResolutionDescriptor{MeridianStep: 0, ParallelStep: 0},
			Responses:         []gll.TransferFunction{{}},
		},
	}
	if _, err := BuildBalloonMesh(def, 0, 40, 1, false); err == nil {
		t.Error("expected error for invalid resolution")
	}
}

func TestBuildBalloonMeshSuccess(t *testing.T) {
	// Use coarse resolution to keep mesh small: 8 meridians × 5 parallels.
	def := makeTestBalloon(45, 45, 90.0)

	m, err := BuildBalloonMesh(def, 0, 40, 1, false)
	if err != nil {
		t.Fatalf("BuildBalloonMesh error: %v", err)
	}
	if len(m.Vertices) == 0 {
		t.Error("expected non-empty vertices")
	}
	if len(m.Vertices) != len(m.Colors) {
		t.Errorf("vertices (%d) and colors (%d) length mismatch", len(m.Vertices), len(m.Colors))
	}
	if len(m.Indices) == 0 {
		t.Error("expected non-empty indices")
	}
	if len(m.Indices)%3 != 0 {
		t.Errorf("indices length %d should be a multiple of 3 for triangles", len(m.Indices))
	}
}

func TestBuildBalloonMeshNormalize(t *testing.T) {
	def := makeTestBalloon(45, 45, 90.0)
	m, err := BuildBalloonMesh(def, 0, 40, 1, true)
	if err != nil {
		t.Fatalf("BuildBalloonMesh(normalize) error: %v", err)
	}
	if len(m.Vertices) == 0 {
		t.Error("expected non-empty vertices for normalize=true")
	}
}

func TestBuildBalloonMeshDefaultsApplied(t *testing.T) {
	// Pass dbRange=0 and scale=0 to exercise default substitution.
	def := makeTestBalloon(45, 45, 90.0)
	m, err := BuildBalloonMesh(def, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("BuildBalloonMesh error: %v", err)
	}
	if len(m.Vertices) == 0 {
		t.Error("expected non-empty vertices")
	}
}

func TestBuildBalloonMeshFreqOutOfRange(t *testing.T) {
	def := makeTestBalloon(45, 45, 90.0)
	if _, err := BuildBalloonMesh(def, 999, 40, 1, false); err == nil {
		t.Error("expected error when no level data exists for freq index")
	}
}

func TestLevelToColorNaN(t *testing.T) {
	c := levelToColor(math.NaN())
	// Expect grey for NaN.
	if c.X != 0.65 || c.Y != 0.65 || c.Z != 0.65 {
		t.Errorf("levelToColor(NaN) = %v, want grey 0.65", c)
	}
}

func TestLevelToColorRange(t *testing.T) {
	cases := []float64{0, 0.25, 0.5, 0.75, 1.0}
	for _, n := range cases {
		c := levelToColor(n)
		if c.X < 0 || c.X > 1 || c.Y < 0 || c.Y > 1 || c.Z < 0 || c.Z > 1 {
			t.Errorf("levelToColor(%v) out of [0,1] range: %v", n, c)
		}
	}
}

func TestHSLToRGBSaturationZero(t *testing.T) {
	c := hslToRGB(0.5, 0, 0.4)
	if c.X != 0.4 || c.Y != 0.4 || c.Z != 0.4 {
		t.Errorf("hslToRGB with s=0 should produce greyscale, got %v", c)
	}
}

func TestHSLToRGBKnownValues(t *testing.T) {
	cases := []struct {
		name    string
		h, s, l float64
		wantR   float64
		wantG   float64
		wantB   float64
	}{
		{"red", 0, 1, 0.5, 1.0, 0.0, 0.0},
		{"green", 1.0 / 3.0, 1, 0.5, 0.0, 1.0, 0.0},
		{"blue", 2.0 / 3.0, 1, 0.5, 0.0, 0.0, 1.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := hslToRGB(tc.h, tc.s, tc.l)
			if math.Abs(c.X-tc.wantR) > 1e-9 || math.Abs(c.Y-tc.wantG) > 1e-9 || math.Abs(c.Z-tc.wantB) > 1e-9 {
				t.Errorf("hslToRGB(%v,%v,%v) = (%v,%v,%v), want (%v,%v,%v)",
					tc.h, tc.s, tc.l, c.X, c.Y, c.Z, tc.wantR, tc.wantG, tc.wantB)
			}
		})
	}
}

func TestHueToRGBBoundaries(t *testing.T) {
	// hueToRGB should handle t < 0 and t > 1 by wrapping.
	r1 := hueToRGB(0, 1, -0.5)
	r2 := hueToRGB(0, 1, 0.5)
	if math.Abs(r1-r2) > 1e-9 {
		t.Errorf("hueToRGB should wrap negative t, got %v vs %v", r1, r2)
	}

	r3 := hueToRGB(0, 1, 1.5)
	r4 := hueToRGB(0, 1, 0.5)
	if math.Abs(r3-r4) > 1e-9 {
		t.Errorf("hueToRGB should wrap t > 1, got %v vs %v", r3, r4)
	}
}
