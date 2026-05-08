package filters

import (
	"math"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// TestBesselAlignment_AllOrders covers every entry in the besselAlignment
// lookup table (orders 1–8 at both 0.5 and 0.25 alignment scales) plus the
// default-fall-through paths (out-of-range order, unsupported scale, the
// outer default branch).
func TestBesselAlignment_AllOrders(t *testing.T) {
	tests := []struct {
		name       string
		order      int
		alignScale float64
		want       float64
	}{
		// 0.5 (3 dB) alignment.
		{"3dB order 1", 1, 0.5, 1.0},
		{"3dB order 2", 2, 0.5, 1.36165412871613},
		{"3dB order 3", 3, 0.5, 1.75567236868121},
		{"3dB order 4", 4, 0.5, 2.11391767490422},
		{"3dB order 5", 5, 0.5, 2.42741070215263},
		{"3dB order 6", 6, 0.5, 2.70339506120292},
		{"3dB order 7", 7, 0.5, 2.95172214703872},
		{"3dB order 8", 8, 0.5, 3.17961723751065},
		{"3dB unsupported order", 9, 0.5, 1.0},
		// 0.25 (6 dB) alignment.
		{"6dB order 1", 1, 0.25, 1.73205080756888},
		{"6dB order 2", 2, 0.25, 1.97694888987955},
		{"6dB order 3", 3, 0.25, 2.42454770439973},
		{"6dB order 4", 4, 0.25, 2.88602284792378},
		{"6dB order 5", 5, 0.25, 3.32415542718002},
		{"6dB order 6", 6, 0.25, 3.72655755891719},
		{"6dB order 7", 7, 0.25, 4.09207415068004},
		{"6dB order 8", 8, 0.25, 4.42556630305568},
		{"6dB unsupported order", 0, 0.25, 1.0},
		// Outer default: any other alignScale (neither 0.5 nor 0.25).
		{"unsupported scale", 2, 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := besselAlignment(tt.order, tt.alignScale)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("besselAlignment(%d, %v) = %v, want %v",
					tt.order, tt.alignScale, got, tt.want)
			}
		})
	}
}

// TestBesselPhaseMatched_AllOrders covers every order in the lookup plus the
// default fall-through.
func TestBesselPhaseMatched_AllOrders(t *testing.T) {
	tests := []struct {
		order int
		want  float64
	}{
		{1, 1.0000000232051},
		{2, 1.73205084237653},
		{3, 2.48134247792628},
		{4, 3.24037034920393},
		{5, 4.00574980621619},
		{6, 4.77560085578494},
		{7, 5.5487473277673},
		{8, 6.32439553519847},
		{0, 1.0},
		{9, 1.0},
		{-1, 1.0},
	}

	for _, tt := range tests {
		got := besselPhaseMatched(tt.order)
		if math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("besselPhaseMatched(%d) = %v, want %v", tt.order, got, tt.want)
		}
	}
}

func TestFrequenciesMatch(t *testing.T) {
	tests := []struct {
		name string
		a, b []float64
		want bool
	}{
		{"both empty", []float64{}, []float64{}, false},
		{"length mismatch", []float64{1, 2}, []float64{1}, false},
		{"exact equal", []float64{1, 2, 3}, []float64{1, 2, 3}, true},
		{"within tolerance", []float64{1000, 2000}, []float64{1000.5, 2000.5}, true},
		{"out of tolerance", []float64{1000, 2000}, []float64{1100, 2100}, false},
		{"NaN in a", []float64{math.NaN(), 1}, []float64{1, 1}, false},
		{"NaN in b", []float64{1, 1}, []float64{1, math.NaN()}, false},
		{"+Inf in a", []float64{math.Inf(1), 1}, []float64{1, 1}, false},
		{"-Inf in b", []float64{1, 1}, []float64{1, math.Inf(-1)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := frequenciesMatch(tt.a, tt.b); got != tt.want {
				t.Errorf("frequenciesMatch(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestFilterResponseStateFinalize_Bypass covers the bypass branch (which
// zeros levels and, separately, zeros phase if hasPhase is true).
func TestFilterResponseStateFinalize_Bypass(t *testing.T) {
	t.Run("bypass with phase", func(t *testing.T) {
		s := &filterResponseState{
			levels:   []float64{10, 20, 30},
			phase:    []float64{1, 2, 3},
			hasPhase: true,
		}
		bank := &gll.GenericFilterBank{ByPass: true}
		s.finalize(bank)
		for i, v := range s.levels {
			if v != 0 {
				t.Errorf("levels[%d] = %v, want 0 (bypass)", i, v)
			}
		}
		for i, v := range s.phase {
			if v != 0 {
				t.Errorf("phase[%d] = %v, want 0 (bypass)", i, v)
			}
		}
	})

	t.Run("bypass without phase leaves phase intact", func(t *testing.T) {
		s := &filterResponseState{
			levels:   []float64{10, 20},
			phase:    []float64{0.1, 0.2},
			hasPhase: false,
		}
		bank := &gll.GenericFilterBank{ByPass: true}
		s.finalize(bank)
		// Levels zeroed.
		if s.levels[0] != 0 || s.levels[1] != 0 {
			t.Errorf("levels = %v, want [0 0]", s.levels)
		}
		// Phase unchanged because hasPhase is false.
		if s.phase[0] != 0.1 || s.phase[1] != 0.2 {
			t.Errorf("phase = %v, want [0.1 0.2]", s.phase)
		}
	})

	t.Run("non-bypass with totalGain applies gain", func(t *testing.T) {
		s := &filterResponseState{
			levels:    []float64{0, 0, 0},
			totalGain: 6.0,
		}
		bank := &gll.GenericFilterBank{}
		s.finalize(bank)
		for i, v := range s.levels {
			if math.Abs(v-6.0) > 1e-12 {
				t.Errorf("levels[%d] = %v, want 6", i, v)
			}
		}
	})

	t.Run("non-bypass with zero totalGain is no-op", func(t *testing.T) {
		s := &filterResponseState{
			levels:    []float64{1, 2, 3},
			totalGain: 0,
		}
		bank := &gll.GenericFilterBank{}
		s.finalize(bank)
		// Levels unchanged.
		want := []float64{1, 2, 3}
		for i, v := range s.levels {
			if v != want[i] {
				t.Errorf("levels[%d] = %v, want %v", i, v, want[i])
			}
		}
	})
}

// TestFilterResponseStateToResult_FIR covers the FIR-shaped result fields
// (SampleRate, IsComplex) which are not exercised by existing LogSpectrum
// happy-path tests.
func TestFilterResponseStateToResult_FIR(t *testing.T) {
	s := &filterResponseState{
		baseFrequencies: []float64{100, 200, 400},
		levels:          []float64{0, -3, -6},
		phase:           []float64{0, 0.1, 0.2},
		hasPhase:        true,
		usedFilters:     2,
		skippedFilters:  1,
		filterKind:      "FIR",
		firSampleRate:   48000,
		firIsComplex:    true,
	}
	bank := &gll.GenericFilterBank{}
	res := s.toResult(bank)

	if res.SampleRate != 48000 {
		t.Errorf("SampleRate = %v, want 48000", res.SampleRate)
	}
	if !res.IsComplex {
		t.Error("IsComplex = false, want true")
	}
	if res.PointCount != 3 {
		t.Errorf("PointCount = %d, want 3", res.PointCount)
	}
	if res.UsedFilters != 2 || res.SkippedFilters != 1 {
		t.Errorf("Used/Skipped = %d/%d, want 2/1", res.UsedFilters, res.SkippedFilters)
	}
	if res.Message == "" {
		t.Error("Message should be populated when skippedFilters>0")
	}
	if len(res.Phase) != 3 {
		t.Errorf("Phase len = %d, want 3 (hasPhase=true)", len(res.Phase))
	}
}

// TestFilterResponseStateToResult_MismatchedAndSkipped exercises the message
// builder branch where both skipped and mismatched are non-zero.
func TestFilterResponseStateToResult_MismatchedAndSkipped(t *testing.T) {
	s := &filterResponseState{
		baseFrequencies:   []float64{100},
		levels:            []float64{0},
		skippedFilters:    2,
		mismatchedFilters: 1,
	}
	res := s.toResult(&gll.GenericFilterBank{})
	if res.Message == "" {
		t.Error("expected non-empty Message")
	}
	// Both fragments should appear.
	if !contains(res.Message, "unsupported") || !contains(res.Message, "mismatched grid") {
		t.Errorf("Message = %q, want to mention both 'unsupported' and 'mismatched grid'", res.Message)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
