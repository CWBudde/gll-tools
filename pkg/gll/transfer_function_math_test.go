package gll

import (
	"math"
	"testing"
)

const (
	dBTolerance          = 1e-9
	phaseTolerance       = 1e-9
	dBRoundTripTolerance = 1e-9
)

// makeTF builds a TransferFunction with `n` bands at `bandsPerOctave`
// resolution starting at `startFreq`, filled with constant level/phase.
func makeTF(bandsPerOctave int32, startFreq float64, n int, level, phase float64) *TransferFunction {
	tf := &TransferFunction{
		Definition: LogSpectrumDefinition{
			BandsPerOctave: bandsPerOctave,
			StartFreq:      startFreq,
			PointCount:     int32(n), //nolint:gosec // n is a small test parameter
		},
		Level: make([]float64, n),
		Phase: make([]float64, n),
	}
	for i := range n {
		tf.Level[i] = level
		tf.Phase[i] = phase
	}
	return tf
}

func TestDefaultAirPropertiesSaneRange(t *testing.T) {
	ap := DefaultAirProperties()
	if ap.Temperature < 0 || ap.Temperature > 40 {
		t.Errorf("Temperature %v outside plausible audio-room range", ap.Temperature)
	}
	if ap.Humidity < 0 || ap.Humidity > 1 {
		t.Errorf("Humidity %v not in [0,1]", ap.Humidity)
	}
	// Speed of sound in air at 20 °C is ~343 m/s.
	if ap.Speed < 330 || ap.Speed > 360 {
		t.Errorf("Speed of sound %v not near 343 m/s", ap.Speed)
	}
	// Standard atmospheric pressure is 101.325 kPa.
	if math.Abs(ap.Pressure-101.325) > 0.5 {
		t.Errorf("Pressure %v not near sea-level standard", ap.Pressure)
	}
}

func TestGetAirLossPerMeterMonotonicAndPositive(t *testing.T) {
	// ISO 9613-1 air absorption is non-negative everywhere in the audio band
	// and (at fixed conditions) strictly increases with frequency across the
	// audio range — high-frequency molecular relaxation dominates.
	ap := DefaultAirProperties()
	freqs := []float64{100, 250, 500, 1000, 2000, 4000, 8000, 16000}
	prev := -1.0
	for _, f := range freqs {
		loss := ap.GetAirLossPerMeter(f)
		if loss < 0 {
			t.Errorf("air loss at %v Hz is negative: %v", f, loss)
		}
		if loss < prev {
			t.Errorf("air loss at %v Hz (%v) decreased from previous (%v)",
				f, loss, prev)
		}
		prev = loss
	}
	// Sanity at the spectral edges.
	if ap.GetAirLossPerMeter(100) > 0.01 {
		t.Errorf("air loss at 100 Hz unexpectedly large: %v dB/m",
			ap.GetAirLossPerMeter(100))
	}
	if ap.GetAirLossPerMeter(16000) <= 0.01 {
		t.Errorf("air loss at 16 kHz unexpectedly small: %v dB/m",
			ap.GetAirLossPerMeter(16000))
	}
}

func TestGetAirLossDefaultsZeroPressure(t *testing.T) {
	// Pressure=0 is documented to fall back to standard sea-level pressure.
	ap0 := AirProperties{Temperature: 20, Humidity: 0.5, Pressure: 0}
	apStd := AirProperties{Temperature: 20, Humidity: 0.5, Pressure: 101.325}
	if math.Abs(ap0.GetAirLossPerMeter(1000)-apStd.GetAirLossPerMeter(1000)) > 1e-12 {
		t.Errorf("zero-pressure should default to standard pressure")
	}
}

func TestCopyDeepIsIndependent(t *testing.T) {
	tf := makeTF(3, 100, 4, 80, 0.25)
	tf.Delay = 0.001

	cpy := tf.CopyDeep()
	cpy.Level[0] = -10
	cpy.Phase[1] = 1.5
	cpy.Delay = 0.999

	if tf.Level[0] != 80 || tf.Phase[1] != 0.25 {
		t.Errorf("CopyDeep is not deep: original level/phase mutated to (%v, %v)",
			tf.Level[0], tf.Phase[1])
	}
	if tf.Delay != 0.001 {
		t.Errorf("CopyDeep is not deep: original delay mutated to %v", tf.Delay)
	}
	if cpy.Definition != tf.Definition {
		t.Errorf("CopyDeep should preserve Definition value")
	}
}

func TestToComplexFromComplexRoundTrip(t *testing.T) {
	cases := []struct {
		level, phase float64
	}{
		{0, 0},                            // unit gain, 0 phase → (1, 0)
		{0, math.Pi / 2},                  // (0, 1)
		{0, -math.Pi / 2},                 // (0, -1)
		{0, math.Pi - 1e-9},               // just shy of π
		{20, 0},                           // 10×, 0 phase
		{-6.020599913279624, math.Pi / 3}, // half-magnitude
		{60, 1.234},                       // arbitrary
	}
	for _, c := range cases {
		tf := makeTF(3, 100, 1, c.level, c.phase)
		re, im := tf.ToComplex()

		expectedMag := math.Pow(10, c.level/20)
		gotMag := math.Hypot(re[0], im[0])
		if math.Abs(gotMag-expectedMag) > expectedMag*1e-12+1e-15 {
			t.Errorf("ToComplex level=%v: magnitude got %v, want %v",
				c.level, gotMag, expectedMag)
		}

		out := makeTF(3, 100, 1, 0, 0)
		out.FromComplex(re, im)
		if math.Abs(out.Level[0]-c.level) > dBRoundTripTolerance {
			t.Errorf("Level round-trip: got %v, want %v", out.Level[0], c.level)
		}
		if math.Abs(out.Phase[0]-c.phase) > phaseTolerance {
			t.Errorf("Phase round-trip: got %v, want %v", out.Phase[0], c.phase)
		}
	}
}

func TestFromComplexZeroMagnitudeFlooredToSentinel(t *testing.T) {
	tf := makeTF(3, 100, 1, 0, 0)
	tf.FromComplex([]float64{0}, []float64{0})
	if math.IsNaN(tf.Level[0]) || math.IsInf(tf.Level[0], 0) {
		t.Errorf("zero magnitude must produce finite sentinel level, got %v",
			tf.Level[0])
	}
	if tf.Level[0] > -100 {
		t.Errorf("zero magnitude should map to a very low level (got %v)",
			tf.Level[0])
	}
}

func TestAddCoherentDoublingIs6dB(t *testing.T) {
	// Two equal in-phase signals at 80 dB sum to 80 + 20·log10(2) ≈ 86.02 dB.
	a := makeTF(3, 100, 4, 80, 0)
	b := makeTF(3, 100, 4, 80, 0)
	a.Add(b)
	want := 80 + 20*math.Log10(2)
	for i, lv := range a.Level {
		if math.Abs(lv-want) > 1e-9 {
			t.Errorf("band %d: got %v dB, want %v dB", i, lv, want)
		}
	}
	for i, ph := range a.Phase {
		if math.Abs(ph) > phaseTolerance {
			t.Errorf("band %d phase: got %v, want 0", i, ph)
		}
	}
}

func TestAddAntiphaseCancellation(t *testing.T) {
	a := makeTF(3, 100, 4, 80, 0)
	b := makeTF(3, 100, 4, 80, math.Pi)
	a.Add(b)
	for i, lv := range a.Level {
		// FromComplex floors zero magnitude at -200 dB.
		if lv > -100 {
			t.Errorf("band %d: anti-phase cancellation should yield very low "+
				"level, got %v dB", i, lv)
		}
	}
}

func TestAddSilenceIsIdentity(t *testing.T) {
	// Adding a vanishingly quiet signal must leave the original essentially
	// unchanged.
	a := makeTF(3, 100, 4, 80, 0.25)
	silent := makeTF(3, 100, 4, -300, 0) // ≈ 1e-15 magnitude
	a.Add(silent)
	for i, lv := range a.Level {
		if math.Abs(lv-80) > 1e-6 {
			t.Errorf("band %d level: %v dB shifted noticeably from 80 dB", i, lv)
		}
		if math.Abs(a.Phase[i]-0.25) > 1e-9 {
			t.Errorf("band %d phase: %v shifted from 0.25", i, a.Phase[i])
		}
	}
}

func TestMultiplyIdentity(t *testing.T) {
	tf := makeTF(3, 100, 4, 80, 0.5)
	identity := makeTF(3, 100, 4, 0, 0)
	tf.Multiply(identity)
	for i := range tf.Level {
		if math.Abs(tf.Level[i]-80) > dBTolerance {
			t.Errorf("band %d: identity multiply shifted level to %v",
				i, tf.Level[i])
		}
		if math.Abs(tf.Phase[i]-0.5) > phaseTolerance {
			t.Errorf("band %d: identity multiply shifted phase to %v",
				i, tf.Phase[i])
		}
	}
}

func TestMultiplyAddsLevelsAndPhases(t *testing.T) {
	tf := makeTF(3, 100, 4, 80, 0.25)
	filter := makeTF(3, 100, 4, 6, math.Pi/4)
	tf.Multiply(filter)
	for i := range tf.Level {
		if math.Abs(tf.Level[i]-86) > dBTolerance {
			t.Errorf("band %d: want 86 dB, got %v", i, tf.Level[i])
		}
		if math.Abs(tf.Phase[i]-(0.25+math.Pi/4)) > phaseTolerance {
			t.Errorf("band %d: phase add wrong", i)
		}
	}
}

func TestMultiplyByPhaseInverter(t *testing.T) {
	// 0 dB / π filter = polarity flip in time domain.
	tf := makeTF(3, 100, 4, 80, 0)
	flip := makeTF(3, 100, 4, 0, math.Pi)
	tf.Multiply(flip)
	for i := range tf.Phase {
		if math.Abs(tf.Phase[i]-math.Pi) > phaseTolerance {
			t.Errorf("band %d: phase should be π, got %v", i, tf.Phase[i])
		}
		if math.Abs(tf.Level[i]-80) > dBTolerance {
			t.Errorf("band %d: level should be unchanged, got %v", i, tf.Level[i])
		}
	}
}

func TestAddGainShiftsLevelOnly(t *testing.T) {
	tf := makeTF(3, 100, 4, 80, 0.5)
	tf.AddGain(6)
	for i := range tf.Level {
		if math.Abs(tf.Level[i]-86) > dBTolerance {
			t.Errorf("band %d: AddGain(6) gave %v, want 86", i, tf.Level[i])
		}
		if math.Abs(tf.Phase[i]-0.5) > phaseTolerance {
			t.Errorf("band %d: phase changed by AddGain", i)
		}
	}
	tf.AddGain(-6)
	for i := range tf.Level {
		if math.Abs(tf.Level[i]-80) > dBTolerance {
			t.Errorf("band %d: AddGain inverse failed: %v", i, tf.Level[i])
		}
	}
}

func TestAddDelayPhaseAtKnownFrequencies(t *testing.T) {
	// 1 band/octave starting at 1000 Hz → bands at 1k, 2k, 4k Hz.
	tf := makeTF(1, 1000, 3, 0, 0)
	const delay = 0.001 // 1 ms
	tf.AddDelay(delay)

	// |Δφ| at frequency f equals |2π f τ|. Sign depends on the e^{±jωt}
	// convention; test magnitude only here.
	for i, f := range []float64{1000, 2000, 4000} {
		want := 2 * math.Pi * f * delay
		if math.Abs(math.Abs(tf.Phase[i])-math.Abs(want)) > phaseTolerance {
			t.Errorf("band %d (f=%v Hz): |phase shift| = %v, want %v",
				i, f, tf.Phase[i], want)
		}
	}
	if math.Abs(tf.Delay-delay) > 1e-15 {
		t.Errorf("Delay accumulator: got %v, want %v", tf.Delay, delay)
	}
}

func TestAddDelayInverseRestoresPhase(t *testing.T) {
	// Self-consistency: applying delay τ then -τ must restore the original
	// phase exactly. Linear in τ on the same frequency grid → no modulo
	// issues.
	tf := makeTF(3, 100, 8, 0, 0.3)
	original := append([]float64(nil), tf.Phase...)
	tf.AddDelay(0.0023)
	tf.AddDelay(-0.0023)
	for i, ph := range tf.Phase {
		if math.Abs(ph-original[i]) > 1e-12 {
			t.Errorf("band %d: phase did not restore (got %v, orig %v)",
				i, ph, original[i])
		}
	}
	if math.Abs(tf.Delay) > 1e-15 {
		t.Errorf("Delay should be 0 after equal-and-opposite delays, got %v",
			tf.Delay)
	}
}

func TestAddDelayLeavesLevelUntouched(t *testing.T) {
	// Pure delay is all-pass: it must not change magnitude at any frequency.
	tf := makeTF(3, 100, 8, 80, 0)
	originalLevel := append([]float64(nil), tf.Level...)
	tf.AddDelay(0.005) // 5 ms
	for i, lv := range tf.Level {
		if math.Abs(lv-originalLevel[i]) > dBTolerance {
			t.Errorf("band %d: AddDelay altered level (%v → %v)",
				i, originalLevel[i], lv)
		}
	}
}

// TestAddDelaySignConvention captures the sign convention used by AddDelay
// and surfaces it explicitly. Engineering DSP convention
// X(f) = ∫ x(t) e^{-j2πft} dt makes a delayed signal multiply its spectrum
// by e^{-j2πfτ}, i.e. phase shifts by -2πfτ. The implementation in
// internal/acoustics/transfer.go uses +2πfτ — verify which sign is in use
// and document it for downstream consumers.
func TestAddDelaySignConvention(t *testing.T) {
	tf := makeTF(1, 1000, 1, 0, 0) // single 1 kHz band, phase 0
	tf.AddDelay(0.00025)           // 0.25 ms → |Δφ| = π/2
	got := tf.Phase[0]
	want := 2 * math.Pi * 1000 * 0.00025
	if math.Abs(got-want) > phaseTolerance {
		t.Errorf("AddDelay sign convention: got phase %v, expected +%v "+
			"(implementation uses phase += 2πfτ). Tighten this test if the "+
			"convention is intentionally e^{-jωτ}.",
			got, want)
	}
}

func TestDirectivityTypeString(t *testing.T) {
	cases := []struct {
		d    DirectivityType
		want string
	}{
		{DirectivityPoint, "Point"},
		{DirectivityLine, "Line"},
		{DirectivityCircularPiston, "CircularPiston"},
		{DirectivityRectangularPiston, "RectangularPiston"},
		{DirectivityType(99), "Unknown"},
		{DirectivityType(-1), "Unknown"},
	}
	for _, c := range cases {
		if got := c.d.String(); got != c.want {
			t.Errorf("DirectivityType(%d).String() = %q, want %q", c.d, got, c.want)
		}
	}
}
