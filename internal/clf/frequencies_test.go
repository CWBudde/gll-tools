package clf

import (
	"math"
	"testing"
)

func TestCLF2FrequenciesLength(t *testing.T) {
	if len(CLF2Frequencies) != 24 {
		t.Errorf("expected 24 CLF2 bands, got %d", len(CLF2Frequencies))
	}
}

func TestCLF1FrequenciesLength(t *testing.T) {
	if len(CLF1Frequencies) != 8 {
		t.Errorf("expected 8 CLF1 bands, got %d", len(CLF1Frequencies))
	}
}

func TestFindNearestFreqIndex(t *testing.T) {
	tests := []struct {
		name   string
		freqs  []float64
		target float64
		want   int
	}{
		{"exact match", []float64{100, 200, 400}, 200, 1},
		{"between values, closer to lower", []float64{100, 200, 400}, 150, 1},
		{"between values, closer to upper", []float64{100, 200, 400}, 350, 2},
		{"below range", []float64{100, 200, 400}, 50, 0},
		{"above range", []float64{100, 200, 400}, 1000, 2},
		{"single element", []float64{1000}, 500, 0},
		{"empty slice", []float64{}, 100, -1},
		{"log distance prefers 1000 over 200 for 500", []float64{200, 1000}, 500, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindNearestFreqIndex(tt.freqs, tt.target)
			if got != tt.want {
				t.Errorf("FindNearestFreqIndex(%v, %v) = %d, want %d", tt.freqs, tt.target, got, tt.want)
			}
		})
	}
}

func TestResampleToBands(t *testing.T) {
	// Build a simple source: 240 points, bandsPerOctave=24, startFreq=20.
	// Level at each point equals its index for easy verification.
	const (
		bandsPerOctave int32   = 24
		startFreq      float64 = 20
		pointCount             = 240
	)

	levels := make([]float64, pointCount)
	for i := range levels {
		levels[i] = float64(i)
	}

	result := ResampleToBands(bandsPerOctave, startFreq, levels, CLF2Frequencies)

	if len(result) != 24 {
		t.Fatalf("expected 24 results, got %d", len(result))
	}

	// Verify the 1000 Hz band maps to the correct source index.
	// 1000 = 20 * 2^(i/24) => i/24 = log2(50) => i = 24*log2(50) ≈ 135.2
	// Nearest integer index = 135.
	expectedIdx := int(math.Round(24 * math.Log2(1000/startFreq)))
	idx1k := 10 // index of 1000 Hz in CLF2Frequencies

	if result[idx1k] != float64(expectedIdx) {
		t.Errorf("1000 Hz band: got level %v, want %v", result[idx1k], float64(expectedIdx))
	}
}
