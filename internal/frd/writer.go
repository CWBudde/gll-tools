// Package frd provides utilities for writing FRD (Frequency Response Data) format files.
// FRD is a simple text format used in professional audio for frequency response data.
package frd

import (
	"fmt"
	"io"
	"math"
)

// WriteResponse writes a single frequency response to an FRD file.
// FRD format: frequency_hz  level_db  phase_deg
// All values are space-separated with no header row.
func WriteResponse(w io.Writer, frequencies, levels, phases []float64) error {
	if len(frequencies) == 0 {
		return fmt.Errorf("no frequency data provided")
	}

	if len(levels) != len(frequencies) {
		return fmt.Errorf("level array length (%d) does not match frequency array length (%d)", len(levels), len(frequencies))
	}

	if len(phases) != len(frequencies) {
		return fmt.Errorf("phase array length (%d) does not match frequency array length (%d)", len(phases), len(frequencies))
	}

	const radToDeg = 180.0 / math.Pi

	for i := range len(frequencies) {
		freq := frequencies[i]
		level := levels[i]
		phaseDeg := phases[i] * radToDeg

		// Format: freq(3 decimals)  level(2 decimals)  phase(2 decimals)
		// Use two spaces between fields for alignment
		_, err := fmt.Fprintf(w, "%.3f  %.2f  %.2f\n", freq, level, phaseDeg)
		if err != nil {
			return fmt.Errorf("failed to write FRD data: %w", err)
		}
	}

	return nil
}
