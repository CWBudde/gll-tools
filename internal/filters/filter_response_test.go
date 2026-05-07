package filters

import (
	"math"
	"testing"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// makeLogSpectrumFile builds a minimal *gll.File with one FilterGroup containing
// a LogSpectrum filter, ready to pass to BuildFilterResponse.
func makeLogSpectrumFile(bands int32, bpo int32, lowestFreq float64) *gll.File {
	levels := make([]float64, bands)
	for i := range levels {
		levels[i] = float64(i) * 0.1
	}
	phase := make([]float64, bands)

	spectrum := &gll.TransferFunctionLP{
		BandsPerOctave:  bpo,
		LowestFrequency: lowestFreq,
		NumberOfBands:   bands,
		Level:           levels,
		Phase:           phase,
	}
	bank := &gll.GenericFilterBank{
		Filters: []gll.GenericBaseFilter{
			{Kind: gll.FilterKindLogSpectrum, LogSpectrum: spectrum},
		},
	}
	return &gll.File{
		Database: &gll.Database{
			FilterGroups: []gll.FilterGroup{
				{
					Label: "TestGroup",
					Key:   "fg1",
					Filters: []gll.FilterDefinition{
						{Label: "TestFilter", Key: "fd1", Filter: bank},
					},
				},
			},
		},
	}
}

func TestBuildFilterResponseNoDatabase(t *testing.T) {
	file := &gll.File{}
	result := BuildFilterResponse(file, FilterResponseRequest{GroupIndex: 0, FilterIndex: 0})
	if result.Success {
		t.Error("expected failure when no database")
	}
}

func TestBuildFilterResponseGroupOutOfRange(t *testing.T) {
	file := makeLogSpectrumFile(48, 6, 125.0)
	result := BuildFilterResponse(file, FilterResponseRequest{GroupIndex: 5, FilterIndex: 0})
	if result.Success {
		t.Error("expected failure for out-of-range group index")
	}
}

func TestBuildFilterResponseFilterOutOfRange(t *testing.T) {
	file := makeLogSpectrumFile(48, 6, 125.0)
	result := BuildFilterResponse(file, FilterResponseRequest{GroupIndex: 0, FilterIndex: 99})
	if result.Success {
		t.Error("expected failure for out-of-range filter index")
	}
}

func TestBuildFilterResponseLogSpectrum(t *testing.T) {
	const bands = 48
	file := makeLogSpectrumFile(bands, 6, 125.0)

	result := BuildFilterResponse(file, FilterResponseRequest{GroupIndex: 0, FilterIndex: 0})
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if len(result.Frequencies) != bands {
		t.Errorf("Frequencies length = %d, want %d", len(result.Frequencies), bands)
	}
	if len(result.Level) != bands {
		t.Errorf("Level length = %d, want %d", len(result.Level), bands)
	}
	for i, f := range result.Frequencies {
		if f <= 0 {
			t.Errorf("Frequencies[%d] = %v, expected > 0", i, f)
		}
	}
}

func TestBuildFilterResponseNilBank(t *testing.T) {
	file := &gll.File{
		Database: &gll.Database{
			FilterGroups: []gll.FilterGroup{
				{
					Label:   "Group",
					Key:     "fg1",
					Filters: []gll.FilterDefinition{{Label: "F", Key: "f1", Filter: nil}},
				},
			},
		},
	}
	result := BuildFilterResponse(file, FilterResponseRequest{GroupIndex: 0, FilterIndex: 0})
	if !result.Success {
		t.Errorf("expected success with nil bank, got error: %s", result.Error)
	}
}

func TestBuildFilterResponseBypassed(t *testing.T) {
	levels := make([]float64, 24)
	for i := range levels {
		levels[i] = -10.0
	}
	spectrum := &gll.TransferFunctionLP{
		BandsPerOctave: 3, LowestFrequency: 100, NumberOfBands: 24, Level: levels,
	}
	bank := &gll.GenericFilterBank{
		ByPass:  true,
		Filters: []gll.GenericBaseFilter{{Kind: gll.FilterKindLogSpectrum, LogSpectrum: spectrum}},
	}
	file := &gll.File{
		Database: &gll.Database{
			FilterGroups: []gll.FilterGroup{
				{Key: "fg1", Filters: []gll.FilterDefinition{{Key: "fd1", Filter: bank}}},
			},
		},
	}
	result := BuildFilterResponse(file, FilterResponseRequest{GroupIndex: 0, FilterIndex: 0})
	if !result.Success {
		t.Fatalf("expected success: %s", result.Error)
	}
	if !result.Bypassed {
		t.Error("expected Bypassed = true")
	}
	for i, v := range result.Level {
		if v != 0 {
			t.Errorf("Level[%d] = %v, want 0 for bypassed filter", i, v)
		}
	}
}

func TestBuildLogSpectrumFrequenciesNil(t *testing.T) {
	freqs := BuildLogSpectrumFrequencies(nil)
	if freqs != nil {
		t.Errorf("expected nil for nil spectrum, got %v", freqs)
	}
}

func TestBuildLogSpectrumFrequenciesZeroBPO(t *testing.T) {
	spec := &gll.TransferFunctionLP{BandsPerOctave: 0, LowestFrequency: 100, NumberOfBands: 10}
	freqs := BuildLogSpectrumFrequencies(spec)
	if freqs != nil {
		t.Error("expected nil for zero BandsPerOctave")
	}
}

func TestBuildLogSpectrumFrequenciesBasic(t *testing.T) {
	spec := &gll.TransferFunctionLP{
		BandsPerOctave:  6,
		LowestFrequency: 125.0,
		NumberOfBands:   6,
		Level:           make([]float64, 6),
	}
	freqs := BuildLogSpectrumFrequencies(spec)
	if len(freqs) != 6 {
		t.Fatalf("expected 6 frequencies, got %d", len(freqs))
	}
	// First frequency should equal LowestFrequency
	if freqs[0] != 125.0 {
		t.Errorf("freqs[0] = %v, want 125.0", freqs[0])
	}
	// Each step should be 2^(1/6) ≈ 1.1225
	ratio := freqs[1] / freqs[0]
	wantRatio := math.Pow(2, 1.0/6.0)
	if math.Abs(ratio-wantRatio) > 1e-9 {
		t.Errorf("frequency ratio = %v, want %v", ratio, wantRatio)
	}
	// All frequencies must be positive and increasing
	for i := 1; i < len(freqs); i++ {
		if freqs[i] <= freqs[i-1] {
			t.Errorf("frequencies not monotonically increasing at index %d", i)
		}
	}
}

func TestBuildLogSpectrumFrequenciesUsesPhaseCountWhenNoLevel(t *testing.T) {
	spec := &gll.TransferFunctionLP{
		BandsPerOctave:  3,
		LowestFrequency: 100,
		NumberOfBands:   12,
		Phase:           make([]float64, 8), // no Level, only Phase
	}
	freqs := BuildLogSpectrumFrequencies(spec)
	if len(freqs) != 8 {
		t.Errorf("expected 8 frequencies (from Phase length), got %d", len(freqs))
	}
}

func TestBuildFilterResponseIIR(t *testing.T) {
	params := &gll.IIRFilterParams{
		FilterType:   gll.FilterTypeLowPass,
		FilterShape:  gll.FilterShapeButterworth,
		Order:        2,
		FreqCritInHz: 1000.0,
		Alignment:    gll.FilterAlignLevel3dB,
	}
	bank := &gll.GenericFilterBank{
		Filters: []gll.GenericBaseFilter{
			{Kind: gll.FilterKindIIR, IIRParams: params},
		},
	}
	file := &gll.File{
		Database: &gll.Database{
			FilterGroups: []gll.FilterGroup{
				{Key: "fg1", Filters: []gll.FilterDefinition{{Key: "fd1", Filter: bank}}},
			},
		},
	}
	result := BuildFilterResponse(file, FilterResponseRequest{GroupIndex: 0, FilterIndex: 0})
	if !result.Success {
		t.Fatalf("IIR filter response failed: %s", result.Error)
	}
	if len(result.Frequencies) == 0 {
		t.Error("expected non-empty frequencies for IIR filter")
	}
	if result.FilterKind != "IIR" {
		t.Errorf("FilterKind = %q, want %q", result.FilterKind, "IIR")
	}
}

func TestMagnitudeToDBInternals(t *testing.T) {
	// magnitudeToDB is exercised through FIR paths in BuildFilterResponse.
	// Provide matching DataDIP (phase) so the filter isn't skipped.
	sampleRate := 48000.0
	nPoints := 10
	irm := make([]float64, nPoints)
	for i := range irm {
		irm[i] = 1.0 // unit magnitude → 0 dB
	}
	dip := make([]float64, nPoints) // phase = 0 radians

	bank := &gll.GenericFilterBank{
		Filters: []gll.GenericBaseFilter{
			{
				Kind: gll.FilterKindFIR,
				FIRData: &gll.FIRFilterData{
					IsTimeResponse: false,
					SampleRate:     sampleRate,
					DataIRM:        irm,
					DataDIP:        dip,
				},
			},
		},
	}
	file := &gll.File{
		Database: &gll.Database{
			FilterGroups: []gll.FilterGroup{
				{Key: "fg1", Filters: []gll.FilterDefinition{{Key: "fd1", Filter: bank}}},
			},
		},
	}
	result := BuildFilterResponse(file, FilterResponseRequest{GroupIndex: 0, FilterIndex: 0})
	if !result.Success {
		t.Fatalf("FIR filter response failed: %s", result.Error)
	}
	if len(result.Level) == 0 {
		t.Error("expected level data for FIR filter")
	}
	// Unit magnitude → 0 dB
	for i, v := range result.Level {
		if math.Abs(v) > 1e-9 {
			t.Errorf("Level[%d] = %v, want ~0 dB for unit magnitude", i, v)
		}
	}
}

func TestBuildFilterResponseSkippedFilters(t *testing.T) {
	// A filter with nil data of a known kind should be skipped
	bank := &gll.GenericFilterBank{
		Filters: []gll.GenericBaseFilter{
			{Kind: gll.FilterKindFIR, FIRData: nil}, // nil data → skipped
		},
	}
	file := &gll.File{
		Database: &gll.Database{
			FilterGroups: []gll.FilterGroup{
				{Key: "fg1", Filters: []gll.FilterDefinition{{Key: "fd1", Filter: bank}}},
			},
		},
	}
	result := BuildFilterResponse(file, FilterResponseRequest{GroupIndex: 0, FilterIndex: 0})
	if !result.Success {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	// No usable data → either skipped or empty response is acceptable.
	// Both branches are valid; we only assert Success above.
	_ = result.SkippedFilters
}
