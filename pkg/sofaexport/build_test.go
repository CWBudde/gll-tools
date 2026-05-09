package sofaexport

import (
	"math"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// fakeBalloon constructs a tiny synthetic balloon: meridian step 90°, parallel
// step 90°. With Symmetry=0 (None) → MeridianCount = 360/90 = 4, ParallelCount =
// 180/90 + 1 = 3. ResponseIndex reserves the first parCount entries for the
// pole strip; subsequent meridians skip the two poles → total = 3 + 3*1 = 6
// stored responses.
func fakeBalloon(level []float64, phase []float64) (*gll.SourceDefinition, *gll.BalloonData) {
	def := gll.LogSpectrumDefinition{BandsPerOctave: 1, StartFreq: 100, PointCount: int32(len(level))} //nolint:gosec // test fixture, length is small
	mkResp := func(scaleDB float64) gll.TransferFunction {
		l := make([]float64, len(level))
		p := make([]float64, len(phase))
		for i := range level {
			l[i] = level[i] + scaleDB
			p[i] = phase[i]
		}
		return gll.TransferFunction{Definition: def, Level: l, Phase: p}
	}

	src := &gll.SourceDefinition{
		Label:            "fake driver",
		MeasuredDistance: 1.0,
		OnAxisSpectrum: &gll.TransferFunction{
			Definition: def,
			Level:      level,
			Phase:      phase,
		},
		OnAxisLevel: 0,
	}

	balloon := &gll.BalloonData{
		AngularResolution: gll.ResolutionDescriptor{
			Symmetry: 0, MeridianStep: 90, ParallelStep: 90,
		},
	}

	// 6 responses: index layout per ResponseIndex with parCount=3, no FHO:
	//   0: (mer=0, par=0) south pole
	//   1: (mer=0, par=1) equator at meridian 0
	//   2: (mer=0, par=2) north pole
	//   3: (mer=1, par=1) equator at meridian 1 (90°)
	//   4: (mer=2, par=1) equator at meridian 2 (180°)
	//   5: (mer=3, par=1) equator at meridian 3 (270°)
	responses := []gll.TransferFunction{
		mkResp(-3),  // south pole, slight loss
		mkResp(0),   // on-axis equator
		mkResp(-3),  // north pole, slight loss
		mkResp(-6),  // 90° off-axis
		mkResp(-12), // behind
		mkResp(-6),  // 270° off-axis
	}
	balloon.Responses = responses
	balloon.ResponseCount = int32(len(responses)) //nolint:gosec // test fixture, length is small
	return src, balloon
}

func TestBuildSOFAFile_Relative(t *testing.T) {
	src, balloon := fakeBalloon([]float64{90, 92}, []float64{0, 0})

	f, err := BuildSOFAFile(src, balloon, BuildContext{Manufacturer: "ACME", Model: "X1"}, Options{Relative: true})
	if err != nil {
		t.Fatalf("BuildSOFAFile: %v", err)
	}

	const merCount, parCount = 4, 3
	if f.M != merCount*parCount || f.R != 1 || f.E != 1 || f.N != 2 {
		t.Errorf("dims = (%d,%d,%d,%d), want (%d,1,1,2)", f.M, f.R, f.E, f.N, merCount*parCount)
	}
	if got, want := len(f.Frequencies), 2; got != want {
		t.Fatalf("len(Frequencies) = %d, want %d", got, want)
	}
	if math.Abs(f.Frequencies[0]-100) > 1e-9 || math.Abs(f.Frequencies[1]-200) > 1e-9 {
		t.Errorf("Frequencies = %v, want [100, 200]", f.Frequencies)
	}

	// Relative mode: the on-axis equator entry (mer=0,par=1) carries the raw
	// balloon level [90, 92] dB → magnitudes 10^(90/20), 10^(92/20).
	row := 0*parCount + 1
	if got := f.TFReal[row][0][0]; math.Abs(got-math.Pow(10, 90.0/20)) > 1e-3 {
		t.Errorf("TFReal[on-axis][0][0] = %g, want %g", got, math.Pow(10, 90.0/20))
	}

	// SourcePosition for on-axis (az=0, el=0) ≈ (1, 0, 0).
	pos := f.SourcePositions[row]
	if math.Abs(pos.X-1) > 1e-9 || math.Abs(pos.Y) > 1e-9 || math.Abs(pos.Z) > 1e-9 {
		t.Errorf("on-axis SourcePosition = %+v, want (1,0,0)", pos)
	}

	// Behind (mer=2 → 180°, par=1 → equator) should be at (-1, 0, 0).
	pos = f.SourcePositions[2*parCount+1]
	if math.Abs(pos.X+1) > 1e-9 || math.Abs(pos.Y) > 1e-9 || math.Abs(pos.Z) > 1e-9 {
		t.Errorf("behind SourcePosition = %+v, want (-1,0,0)", pos)
	}

	if f.Title == "" {
		t.Error("Title is empty")
	}
	if f.SOFAConventions != "FreeFieldDirectivityTF" {
		t.Errorf("SOFAConventions = %q", f.SOFAConventions)
	}
	if f.DataType != "TF" {
		t.Errorf("DataType = %q", f.DataType)
	}
}

func TestBuildSOFAFile_Combined(t *testing.T) {
	// Balloon level=0 (relative directivity flat) so combined ≡ on-axis only.
	src, balloon := fakeBalloon([]float64{90, 92}, []float64{0, 0})
	// Replace balloon levels with zeros so the directivity is flat 0 dB.
	for i := range balloon.Responses {
		for j := range balloon.Responses[i].Level {
			balloon.Responses[i].Level[j] = 0
		}
	}

	f, err := BuildSOFAFile(src, balloon, BuildContext{}, Options{})
	if err != nil {
		t.Fatalf("BuildSOFAFile: %v", err)
	}

	const parCount = 3
	row := 0*parCount + 1 // on-axis
	want := math.Pow(10, 90.0/20)
	if got := f.TFReal[row][0][0]; math.Abs(got-want) > 1e-3 {
		t.Errorf("combined on-axis TFReal[0] = %g, want %g (= 10^(90/20))", got, want)
	}
}
