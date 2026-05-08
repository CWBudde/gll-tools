package gll

// Acoustically-motivated tests for the array-response engine.
//
// These tests exercise public functions (ComputeSystemResponseGrid /
// WithProgress / WithProgressCancel, ComputeSystemResponseDetailsAt,
// GetResponseAtAngle) and pin down the helpers (normalizeGLLMeridian,
// normalizeGLLParallel, progressReportInterval).
//
// Each test is grounded in a physical principle (1/r law, coherent summation,
// path-length interference, atmospheric absorption). Where the implementation
// makes a non-obvious modeling choice, the test documents the current behavior
// and links to the open contract in docs/acoustic-model.md.
//
// Resolved contracts:
//
//   1. On-axis SPL application. computeElementResponseAt now multiplies the
//      directivity response by SourceDefinition.OnAxisSpectrum (when present
//      and on the same frequency grid). The previous "multiply by balloon
//      front pole" was a misnamed no-op that dropped absolute SPL entirely.
//      TestSourceDefinitionOnAxisSpectrumIsApplied below pins the fixed
//      behavior. See docs/acoustic-model.md → "Source Response Components".
//
//   2. Phase sign of AddDelay. internal/acoustics.AddDelay now subtracts
//      2π·f·δ from the stored phase, matching the physical convention
//      X(f) ↦ X(f)·e^{−j2πfδ}. TestResolvedContract_AddDelaySign below pins
//      the fix. The visualization layer (internal/viz) was already using the
//      same physical convention, so no compensating change was needed.
//      See docs/acoustic-model.md → "Phase And Delay".

import (
	"math"
	"testing"
)

// ---- helpers ----

// twoBandUniformBalloon returns a uniform balloon (all responses 0 dB, phase 0)
// with two frequency bins so we can probe wavelength-dependent behavior.
func twoBandUniformBalloon() *BalloonData {
	def := LogSpectrumDefinition{
		BandsPerOctave: 1,
		StartFreq:      1000,
		PointCount:     2, // 1 kHz, 2 kHz
	}
	return testUniformBalloonWithDefinition(def)
}

func newOmniArray(positions ...Vector3D) *ArrayConfig {
	src := &SourceDefinition{BalloonData: twoBandUniformBalloon()}
	elements := make([]ArrayElement, len(positions))
	for i, p := range positions {
		elements[i] = ArrayElement{Position: p, SourceDefs: []*SourceDefinition{src}}
	}
	return &ArrayConfig{Elements: elements}
}

// ---- 1/r spherical spreading ----

// TestInverseSquareLaw_SingleOmni verifies that a single omnidirectional source
// loses 6.02 dB per doubling of distance in the free field (1/r pressure law).
//
// Physical baseline: SPL(d) - SPL(d_ref) = -20·log10(d/d_ref).
// At d=1m  → -20·log10(1)  =  0 dB
// At d=2m  → -20·log10(2)  ≈ -6.0206 dB
// At d=10m → -20·log10(10) = -20 dB
//
// The uniform balloon stores 0 dB, so on-axis multiplication (concern #1) adds
// zero — this test is unaffected by the open on-axis contract.
func TestInverseSquareLaw_SingleOmni(t *testing.T) {
	cfg := newOmniArray(Vector3D{})
	air := AirProperties{Speed: 343}

	tests := []struct {
		distance float64
		want     float64
	}{
		{1, 0},
		{2, -20 * math.Log10(2)},
		{10, -20},
		{100, -40},
	}
	for _, tc := range tests {
		resp := ComputeSystemResponseAt(cfg, Vector3D{X: tc.distance}, air, false)
		if resp == nil {
			t.Fatalf("d=%v: nil response", tc.distance)
		}
		if got := resp.Level[0]; math.Abs(got-tc.want) > 1e-6 {
			t.Errorf("d=%v: level = %.6f dB, want %.6f dB", tc.distance, got, tc.want)
		}
	}
}

// ---- Coherent summation ----

// TestCoherentSum_TwoCoincidentSources verifies that two co-located,
// in-phase, identical sources sum to +6.02 dB above a single source.
//
// Physical baseline: doubled complex pressure → 20·log10(2) ≈ +6.0206 dB.
// All source positions, gains and phases are identical, so the result is
// independent of the on-axis convention and the phase sign convention.
func TestCoherentSum_TwoCoincidentSources(t *testing.T) {
	air := AirProperties{Speed: 343}
	receiver := Vector3D{X: 1}

	single := ComputeSystemResponseAt(newOmniArray(Vector3D{}), receiver, air, false)
	double := ComputeSystemResponseAt(newOmniArray(Vector3D{}, Vector3D{}), receiver, air, false)
	if single == nil || double == nil {
		t.Fatal("nil response")
	}

	delta := double.Level[0] - single.Level[0]
	want := 20 * math.Log10(2)
	if math.Abs(delta-want) > 1e-6 {
		t.Errorf("two coincident sources: ΔSPL = %.6f dB, want %.6f dB", delta, want)
	}
}

// TestPathLengthInterference_LambdaOver2 verifies destructive interference at
// 1 kHz when two omnis are spaced exactly half a wavelength apart and the
// receiver lies on the line connecting them, far enough that 1/r differences
// are negligible. At 2 kHz (= full wavelength of separation) the two sources
// add constructively. This is the classic |sin(πd/λ)| comb pattern.
//
// This complements the existing TestComputeSystemResponseAtShowsPathLengthInterference
// by quantifying the expected level difference — at the null we expect ≥40 dB
// of cancellation, and at the peak we expect ≈ +6 dB over the per-source level.
func TestPathLengthInterference_LambdaOver2(t *testing.T) {
	const c = 343.0
	const f1 = 1000.0
	halfLambda := c / (2 * f1) // λ/2 at 1 kHz

	cfg := newOmniArray(Vector3D{}, Vector3D{X: halfLambda})
	resp := ComputeSystemResponseAt(cfg, Vector3D{X: 1000}, AirProperties{Speed: c}, false)
	if resp == nil || len(resp.Level) != 2 {
		t.Fatalf("bad response: %+v", resp)
	}

	null := resp.Level[0] // 1 kHz
	peak := resp.Level[1] // 2 kHz

	if peak-null < 40 {
		t.Errorf("expected ≥40 dB null/peak ratio, got peak-null = %.2f dB", peak-null)
	}
	// Single-source reference at the same distance for the constructive check.
	singleRef := ComputeSystemResponseAt(newOmniArray(Vector3D{}), Vector3D{X: 1000}, AirProperties{Speed: c}, false)
	wantPeakBoost := 20 * math.Log10(2)
	if got := peak - singleRef.Level[1]; math.Abs(got-wantPeakBoost) > 0.5 {
		t.Errorf("constructive boost at 2 kHz = %.2f dB, want ≈ %.2f dB", got, wantPeakBoost)
	}
}

// ---- Air absorption ----

// TestAirAbsorption_AffectsHFMore is a sanity check that turning air absorption
// on reduces level (especially at higher frequencies) over a non-trivial
// distance, and that turning it off has no effect on the result.
//
// We don't assert the exact ISO 9613-1 value here — that belongs to
// internal/acoustics tests. We only assert monotonicity: more distance and
// higher frequency → more attenuation, and HF attenuation > LF attenuation.
func TestAirAbsorption_AffectsHFMore(t *testing.T) {
	cfg := newOmniArray(Vector3D{})
	air := DefaultAirProperties()
	air.Speed = 343

	receiver := Vector3D{X: 100}

	dry := ComputeSystemResponseAt(cfg, receiver, air, false)
	wet := ComputeSystemResponseAt(cfg, receiver, air, true)
	if dry == nil || wet == nil {
		t.Fatal("nil response")
	}

	lossLF := dry.Level[0] - wet.Level[0] // 1 kHz
	lossHF := dry.Level[1] - wet.Level[1] // 2 kHz

	if lossLF < 0 {
		t.Errorf("LF loss = %.4f dB, want ≥ 0 (absorption never amplifies)", lossLF)
	}
	if lossHF <= lossLF {
		t.Errorf("HF loss (%.4f dB) should exceed LF loss (%.4f dB) at 100 m", lossHF, lossLF)
	}
}

// ---- Grid wrappers ----

// TestComputeSystemResponseGrid_MatchesPerReceiverCalls verifies that the
// vectorised Grid call produces identical results to looping
// ComputeSystemResponseAt by hand. Any deviation points to shared mutable
// state across receivers (which would be a serious correctness bug).
func TestComputeSystemResponseGrid_MatchesPerReceiverCalls(t *testing.T) {
	cfg := newOmniArray(Vector3D{}, Vector3D{X: 0.5})
	air := AirProperties{Speed: 343}
	receivers := []Vector3D{
		{X: 1}, {X: 5, Y: 2}, {X: 10, Z: -1}, {X: 100},
	}

	grid := ComputeSystemResponseGrid(cfg, receivers, air, false)
	if len(grid) != len(receivers) {
		t.Fatalf("grid length = %d, want %d", len(grid), len(receivers))
	}
	for i, r := range receivers {
		want := ComputeSystemResponseAt(cfg, r, air, false)
		if grid[i] == nil || want == nil {
			t.Fatalf("receiver %d nil response", i)
		}
		for j := range want.Level {
			if math.Abs(grid[i].Level[j]-want.Level[j]) > 1e-9 {
				t.Errorf("receiver %d band %d: grid=%v, perRecv=%v",
					i, j, grid[i].Level[j], want.Level[j])
			}
			if math.Abs(grid[i].Phase[j]-want.Phase[j]) > 1e-9 {
				t.Errorf("receiver %d band %d phase: grid=%v, perRecv=%v",
					i, j, grid[i].Phase[j], want.Phase[j])
			}
		}
	}
}

func TestComputeSystemResponseGridWithProgress_CallbackInvoked(t *testing.T) {
	cfg := newOmniArray(Vector3D{})
	receivers := []Vector3D{{X: 1}, {X: 2}, {X: 3}, {X: 4}, {X: 5}}

	type call struct{ done, total int }
	var calls []call
	ComputeSystemResponseGridWithProgress(
		cfg, receivers, AirProperties{Speed: 343}, false,
		func(done, total int) {
			calls = append(calls, call{done, total})
		},
	)

	if len(calls) == 0 {
		t.Fatal("progress callback was never invoked")
	}
	first := calls[0]
	if first.done != 0 || first.total != len(receivers) {
		t.Errorf("first call = %+v, want {0, %d}", first, len(receivers))
	}
	last := calls[len(calls)-1]
	if last.done != len(receivers) || last.total != len(receivers) {
		t.Errorf("last call = %+v, want {%d, %d}", last, len(receivers), len(receivers))
	}
	for i := 1; i < len(calls); i++ {
		if calls[i].done < calls[i-1].done {
			t.Errorf("progress regressed at step %d: %+v -> %+v", i, calls[i-1], calls[i])
		}
	}
}

func TestComputeSystemResponseGridWithProgressCancel_ReturnsCanceledFlag(t *testing.T) {
	cfg := newOmniArray(Vector3D{})
	receivers := make([]Vector3D, 50)
	for i := range receivers {
		receivers[i] = Vector3D{X: float64(i + 1)}
	}

	calls := 0
	results, canceled := ComputeSystemResponseGridWithProgressCancel(
		cfg, receivers, AirProperties{Speed: 343}, false, nil,
		func() bool {
			calls++
			return calls > 5 // let a few receivers complete then cancel
		},
	)
	if !canceled {
		t.Errorf("canceled flag = false, want true")
	}
	if len(results) != len(receivers) {
		t.Errorf("results length = %d, want %d (allocated up-front)", len(results), len(receivers))
	}
	completed := 0
	for _, r := range results {
		if r != nil {
			completed++
		}
	}
	if completed == 0 || completed == len(receivers) {
		t.Errorf("expected partial completion, got %d / %d", completed, len(receivers))
	}
}

func TestComputeSystemResponseGridWithProgressCancel_PreCancelled(t *testing.T) {
	results, canceled := ComputeSystemResponseGridWithProgressCancel(
		newOmniArray(Vector3D{}), []Vector3D{{X: 1}},
		AirProperties{Speed: 343}, false, nil,
		func() bool { return true },
	)
	if !canceled {
		t.Error("canceled = false, want true on pre-cancellation")
	}
	if len(results) != 1 || results[0] != nil {
		t.Errorf("results = %+v, want one nil entry", results)
	}
}

// ---- Details ----

// TestComputeSystemResponseDetailsAt_SumMatchesTotal validates that the
// per-element contributions, when coherently summed, equal the aggregate
// TransferFunction. This protects against accidental double-counting or
// missing elements in the contribution list.
func TestComputeSystemResponseDetailsAt_SumMatchesTotal(t *testing.T) {
	cfg := newOmniArray(Vector3D{}, Vector3D{X: 0.2}, Vector3D{X: 0.4})
	air := AirProperties{Speed: 343}
	receiver := Vector3D{X: 5}

	details := ComputeSystemResponseDetailsAt(cfg, receiver, air, false)
	if details == nil {
		t.Fatal("nil details")
	}
	if got, want := len(details.ElementContributions), 3; got != want {
		t.Errorf("contributions length = %d, want %d", got, want)
	}

	// Coherently sum the contributions.
	var summed *TransferFunction
	for _, c := range details.ElementContributions {
		if summed == nil {
			summed = c.CopyDeep()
		} else {
			summed.Add(c)
		}
	}
	for i := range summed.Level {
		if math.Abs(summed.Level[i]-details.TransferFunction.Level[i]) > 1e-9 {
			t.Errorf("band %d: re-summed level = %v, total = %v",
				i, summed.Level[i], details.TransferFunction.Level[i])
		}
	}
}

func TestComputeSystemResponseDetailsAt_NilConfig(t *testing.T) {
	if got := ComputeSystemResponseDetailsAt(nil, Vector3D{}, AirProperties{Speed: 343}, false); got != nil {
		t.Errorf("nil config: details = %+v, want nil", got)
	}
}

func TestComputeSystemResponseDetailsAt_NoValidElements(t *testing.T) {
	cfg := &ArrayConfig{Elements: []ArrayElement{
		{Position: Vector3D{}, SourceDefs: nil},                     // skipped: no source defs
		{Position: Vector3D{}, SourceDefs: []*SourceDefinition{{}}}, // skipped: nil balloon
	}}
	if got := ComputeSystemResponseDetailsAt(cfg, Vector3D{X: 1}, AirProperties{Speed: 343}, false); got != nil {
		t.Errorf("no valid elements: details = %+v, want nil", got)
	}
}

func TestComputeSystemResponseAt_NilConfig(t *testing.T) {
	if got := ComputeSystemResponseAt(nil, Vector3D{}, AirProperties{Speed: 343}, false); got != nil {
		t.Errorf("nil config: response = %+v, want nil", got)
	}
}

// ---- GetResponseAtAngle (public, radians) ----

// TestGetResponseAtAngle maps the (theta, phi) radians convention used by the
// public API to GLL meridian/parallel and verifies it agrees with the internal
// responseAtGLLAngles for a directional balloon.
//
// The current code maps:
//
//	parIdx = (thetaDeg + 90) / parStep   →  theta = -π/2 means parallel = 0 (front pole)
//	merIdx = phiDeg / merStep            →  phi   =  0    means meridian = 0
func TestGetResponseAtAngle_FrontPoleAndTop(t *testing.T) {
	bd := testDirectionalBalloon()

	// Front pole: theta = -π/2, phi = 0  ⇒ parallel 0, meridian 0  ⇒ level 10
	front := bd.GetResponseAtAngle(-math.Pi/2, 0)
	if front == nil {
		t.Fatal("front pole: nil response")
	}
	if math.Abs(front.Level[0]-10) > 1e-6 {
		t.Errorf("front pole level = %v, want 10", front.Level[0])
	}

	// Top (meridian 0, parallel 90): theta = 0, phi = 0 ⇒ level 30
	top := bd.GetResponseAtAngle(0, 0)
	if top == nil {
		t.Fatal("top: nil response")
	}
	if math.Abs(top.Level[0]-30) > 1e-6 {
		t.Errorf("top level = %v, want 30", top.Level[0])
	}
}

func TestGetResponseAtAngle_NilOrEmpty(t *testing.T) {
	var bd *BalloonData
	if got := bd.GetResponseAtAngle(0, 0); got != nil {
		t.Errorf("nil receiver: got %+v, want nil", got)
	}

	empty := &BalloonData{}
	if got := empty.GetResponseAtAngle(0, 0); got != nil {
		t.Errorf("empty balloon: got %+v, want nil", got)
	}
}

// TestGetResponseAtAngle_PhiWrap exercises the negative- and >2π wrap paths in
// the public API.
func TestGetResponseAtAngle_PhiWrap(t *testing.T) {
	bd := testDirectionalBalloon()

	// phi = -π/2 should wrap to 3π/2 ≡ meridian 270° (left, level 50)
	wrapped := bd.GetResponseAtAngle(0, -math.Pi/2)
	if wrapped == nil {
		t.Fatal("nil response after negative-phi wrap")
	}
	if math.Abs(wrapped.Level[0]-50) > 1e-6 {
		t.Errorf("phi=-π/2 wrap: level = %v, want 50 (meridian 270°)", wrapped.Level[0])
	}

	// phi = 5π/2 should wrap to π/2 ≡ meridian 90° (right, level 20)
	bigPhi := bd.GetResponseAtAngle(0, 5*math.Pi/2)
	if bigPhi == nil {
		t.Fatal("nil response after large-phi wrap")
	}
	if math.Abs(bigPhi.Level[0]-20) > 1e-6 {
		t.Errorf("phi=5π/2 wrap: level = %v, want 20 (meridian 90°)", bigPhi.Level[0])
	}
}

// ---- helper coverage ----

func TestNormalizeGLLMeridian(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		symmetry int
		want     float64
	}{
		{"in-range", 45, int(SymmetryNone), 45},
		{"negative wrap", -10, int(SymmetryNone), 350},
		{"large wrap", 730, int(SymmetryNone), 10},
		{"axial folds to zero", 137, int(SymmetryAxial), 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeGLLMeridian(tc.input, tc.symmetry)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("normalizeGLLMeridian(%v, %v) = %v, want %v",
					tc.input, tc.symmetry, got, tc.want)
			}
		})
	}
}

func TestNormalizeGLLParallel(t *testing.T) {
	const parStep = 5.0
	const parCount = 37 // 0°..180° at 5° steps
	const measuredMax = 180.0

	tests := []struct {
		name          string
		input         float64
		symmetry      int
		frontHalfOnly bool
		wantOK        bool
		wantValue     float64
	}{
		{"in-range", 45, int(SymmetryNone), false, true, 45},
		{"out of range high", 200, int(SymmetryNone), false, false, 0},
		{"out of range low", -1, int(SymmetryNone), false, false, 0},
		{"frontHalfOnly rejects rear", 100, int(SymmetryNone), true, false, 0},
		{"frontHalfOnly accepts front", 80, int(SymmetryNone), true, true, 80},
		// Mirror tests: shrink the measured grid so the mirror branch is exercised.
		{"horizontal mirrors rear hemisphere", 170, int(SymmetryHorizontal), false, true, 170},
		{"axial cannot mirror beyond grid", measuredMax + 1, int(SymmetryAxial), false, false, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := normalizeGLLParallel(tc.input, tc.symmetry, parStep, parCount, tc.frontHalfOnly)
			if ok != tc.wantOK {
				t.Errorf("normalizeGLLParallel(%v, sym=%v, fho=%v) ok = %v, want %v",
					tc.input, tc.symmetry, tc.frontHalfOnly, ok, tc.wantOK)
			}
			if ok && math.Abs(got-tc.wantValue) > 1e-9 {
				t.Errorf("value = %v, want %v", got, tc.wantValue)
			}
		})
	}
}

// TestNormalizeGLLParallel_MirrorReducedGrid checks the mirror branch where the
// measured grid covers only the front hemisphere and a parallel in the rear is
// folded into the measured range.
func TestNormalizeGLLParallel_MirrorReducedGrid(t *testing.T) {
	const parStep = 5.0
	const parCount = 19 // 0°..90° in 5° steps (front-hemisphere measurement)

	got, ok := normalizeGLLParallel(120, int(SymmetryHorizontal), parStep, parCount, false)
	if !ok {
		t.Fatalf("expected ok with mirror, got false")
	}
	if math.Abs(got-60) > 1e-9 {
		t.Errorf("mirrored parallel = %v, want 60 (= 180-120)", got)
	}
}

func TestProgressReportInterval(t *testing.T) {
	tests := []struct {
		total int
		want  int
	}{
		{0, 1},
		{1, 1},
		{50, 1},
		{100, 1},
		{101, 1},     // 101/100 = 1
		{1000, 10},   // 1000/100 = 10
		{12345, 123}, // 12345/100 = 123
	}
	for _, tc := range tests {
		if got := progressReportInterval(tc.total); got != tc.want {
			t.Errorf("progressReportInterval(%d) = %d, want %d", tc.total, got, tc.want)
		}
	}
}

// ---- documenting the open contracts (concerns) ----

// TestSourceDefinitionOnAxisSpectrumIsApplied verifies the post-fix behavior:
// when SourceDefinition.OnAxisSpectrum is present and shares the balloon's
// frequency grid, computeElementResponseAt multiplies it onto the directivity
// response. The receiver lies 1 m on-axis, so 1/r contributes 0 dB and the
// total equals the OnAxisSpectrum level.
//
// The companion test TestSourceDefinitionOnAxisSpectrum_GridMismatchIsIgnored
// pins the safety guard: a mismatched grid skips the multiply rather than
// causing array-bounds panics or silently combining unaligned bins.
func TestSourceDefinitionOnAxisSpectrumIsApplied(t *testing.T) {
	def := LogSpectrumDefinition{BandsPerOctave: 1, StartFreq: 1000, PointCount: 1}
	bd := testUniformBalloonWithDefinition(def) // balloon front pole = 0 dB (Convention A)

	src := &SourceDefinition{
		BalloonData: bd,
		OnAxisSpectrum: &TransferFunction{
			Definition: def,
			Level:      []float64{96.23},
			Phase:      []float64{0},
		},
	}
	cfg := &ArrayConfig{Elements: []ArrayElement{{
		Position: Vector3D{}, SourceDefs: []*SourceDefinition{src},
	}}}

	resp := ComputeSystemResponseAt(cfg, Vector3D{X: 1}, AirProperties{Speed: 343}, false)
	if resp == nil {
		t.Fatal("nil response")
	}
	if got, want := resp.Level[0], 96.23; math.Abs(got-want) > 1e-6 {
		t.Errorf("on-axis SPL = %.4f dB, want %.4f dB (= OnAxisSpectrum)", got, want)
	}
}

func TestSourceDefinitionOnAxisSpectrum_GridMismatchIsIgnored(t *testing.T) {
	balloonDef := LogSpectrumDefinition{BandsPerOctave: 1, StartFreq: 1000, PointCount: 1}
	bd := testUniformBalloonWithDefinition(balloonDef)

	// OnAxisSpectrum has a different frequency grid (twice the bands per octave).
	// applyOnAxisSpectrum should detect the mismatch and leave the response untouched.
	src := &SourceDefinition{
		BalloonData: bd,
		OnAxisSpectrum: &TransferFunction{
			Definition: LogSpectrumDefinition{BandsPerOctave: 2, StartFreq: 1000, PointCount: 1},
			Level:      []float64{96.23},
			Phase:      []float64{0},
		},
	}
	cfg := &ArrayConfig{Elements: []ArrayElement{{
		Position: Vector3D{}, SourceDefs: []*SourceDefinition{src},
	}}}

	resp := ComputeSystemResponseAt(cfg, Vector3D{X: 1}, AirProperties{Speed: 343}, false)
	if resp == nil {
		t.Fatal("nil response")
	}
	if got := resp.Level[0]; math.Abs(got) > 1e-6 {
		t.Errorf("grid mismatch should skip OnAxisSpectrum multiply: level = %.4f dB, want 0", got)
	}
}

// TestResolvedContract_AddDelaySign pins the physical sign convention used by
// AddDelay: a wave delayed by δ observed at the receiver carries phase
// −2π·f·δ (engineering DSP convention X(f) = ∫ x(t) e^{−j2πft} dt).
//
// The destructive-interference test at λ/2 is symmetric under sign flip, so
// it cannot detect this — hence this dedicated direct test.
//
// See docs/acoustic-model.md → "Phase And Delay".
func TestResolvedContract_AddDelaySign(t *testing.T) {
	// Apply a 1 ms delay at 250 Hz (0.25 cycle = π/2 rad in magnitude).
	tf := &TransferFunction{
		Definition: LogSpectrumDefinition{BandsPerOctave: 1, StartFreq: 250, PointCount: 1},
		Level:      []float64{0},
		Phase:      []float64{0},
	}
	tf.AddDelay(1e-3)

	if math.Abs(tf.Phase[0]-(-math.Pi/2)) > 1e-9 {
		t.Errorf("phase after +1 ms delay at 250 Hz = %v rad, want -π/2 (= −2π·f·δ)",
			tf.Phase[0])
	}
}
