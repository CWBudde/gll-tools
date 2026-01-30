// Package clf provides utilities for CLF (Common Loudspeaker Format) export.
package clf

import "math"

// CLF2Frequencies contains the 24 third-octave bands used by CLF2 (100 Hz to 20000 Hz).
var CLF2Frequencies = []float64{
	100, 125, 160, 200, 250, 315, 400, 500, 630, 800,
	1000, 1250, 1600, 2000, 2500, 3150, 4000, 5000, 6300, 8000,
	10000, 12500, 16000, 20000,
}

// CLF1Frequencies contains the 8 octave bands used by CLF1.
var CLF1Frequencies = []float64{
	125, 250, 500, 1000, 2000, 4000, 8000, 16000,
}

// FindNearestFreqIndex returns the index of the frequency in freqs closest
// to target, using logarithmic distance.
func FindNearestFreqIndex(freqs []float64, target float64) int {
	if len(freqs) == 0 {
		return -1
	}

	logTarget := math.Log2(target)
	bestIdx := 0
	bestDist := math.Abs(math.Log2(freqs[0]) - logTarget)

	for i := 1; i < len(freqs); i++ {
		dist := math.Abs(math.Log2(freqs[i]) - logTarget)
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}

	return bestIdx
}

// ResampleToBands resamples GLL transfer function levels to target frequency
// bands using nearest-neighbor interpolation on a logarithmic scale.
//
// The GLL frequency grid is defined as: freq[i] = startFreq * 2^(i/bandsPerOctave).
func ResampleToBands(bandsPerOctave int32, startFreq float64, levels []float64, targetFreqs []float64) []float64 {
	result := make([]float64, len(targetFreqs))

	// Build the source frequency list.
	srcFreqs := make([]float64, len(levels))
	for i := range levels {
		srcFreqs[i] = startFreq * math.Pow(2, float64(i)/float64(bandsPerOctave))
	}

	for i, target := range targetFreqs {
		idx := FindNearestFreqIndex(srcFreqs, target)
		if idx >= 0 && idx < len(levels) {
			result[i] = levels[idx]
		}
	}

	return result
}
