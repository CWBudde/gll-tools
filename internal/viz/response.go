package viz

import (
	"errors"
	"math"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

type ResponseSeries struct {
	Frequencies  []float64
	Level        []float64
	Phase        []float64
	PhaseWrapped []float64
	GroupDelayMs []float64
	UsesOnAxis   bool
}

func BuildResponseSeries(def *gll.SourceDefinition, response *gll.TransferFunction, combineOnAxis bool) (*ResponseSeries, error) {
	// Validate response and build frequency grid
	if response == nil {
		return nil, errors.New("missing response")
	}
	freqs := BuildFrequencyList(response.Definition)
	if len(freqs) == 0 {
		return nil, errors.New("invalid response frequency grid")
	}

	// Copy level/phase to avoid mutating source
	levels := append([]float64(nil), response.Level...)
	phase := append([]float64(nil), response.Phase...)

	// Optionally combine on-axis spectrum
	usesOnAxis := false
	if combineOnAxis && def != nil && def.OnAxisSpectrum != nil && len(def.OnAxisSpectrum.Level) > 0 {
		if sameSpectrumGrid(def.OnAxisSpectrum.Definition, response.Definition) {
			usesOnAxis = true
			for i := range levels {
				if i < len(def.OnAxisSpectrum.Level) {
					levels[i] += def.OnAxisSpectrum.Level[i]
				}
			}
			if len(phase) > 0 && len(def.OnAxisSpectrum.Phase) == len(phase) {
				for i := range phase {
					phase[i] += def.OnAxisSpectrum.Phase[i]
				}
			}
		}
	}

	// Apply delay, unwrap, and derive helper series
	phase = applyDelayToPhase(phase, freqs, response.Delay)
	phase = unwrapPhase(phase)
	phaseWrapped := wrapPhaseSlice(phase)
	groupDelay := computeGroupDelayMs(freqs, phase)

	return &ResponseSeries{
		Frequencies:  freqs,
		Level:        levels,
		Phase:        phase,
		PhaseWrapped: phaseWrapped,
		GroupDelayMs: groupDelay,
		UsesOnAxis:   usesOnAxis,
	}, nil
}

func applyDelayToPhase(phase, frequencies []float64, delaySeconds float64) []float64 {
	// Subtract phase advance for delay
	if delaySeconds == 0 || len(phase) == 0 || len(phase) != len(frequencies) {
		return phase
	}
	factor := 2 * math.Pi * delaySeconds
	out := make([]float64, len(phase))
	for i := range phase {
		out[i] = phase[i] - frequencies[i]*factor
	}
	return out
}

func unwrapPhase(phase []float64) []float64 {
	// Unwrap phase by removing 2π jumps
	if len(phase) == 0 {
		return phase
	}
	out := make([]float64, len(phase))
	out[0] = phase[0]
	offset := 0.0
	for i := 1; i < len(phase); i++ {
		delta := phase[i] - phase[i-1]
		if delta > math.Pi {
			offset -= 2 * math.Pi
		} else if delta < -math.Pi {
			offset += 2 * math.Pi
		}
		out[i] = phase[i] + offset
	}
	return out
}

func wrapPhase(value float64) float64 {
	// Wrap single phase value to [-π, π]
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return math.NaN()
	}
	twoPi := 2 * math.Pi
	wrapped := math.Mod(value+math.Pi, twoPi)
	if wrapped < 0 {
		wrapped += twoPi
	}
	return wrapped - math.Pi
}

func wrapPhaseSlice(values []float64) []float64 {
	// Wrap a slice of phase values
	if len(values) == 0 {
		return values
	}
	out := make([]float64, len(values))
	for i, v := range values {
		out[i] = wrapPhase(v)
	}
	return out
}

func computeGroupDelayMs(frequencies, phaseUnwrapped []float64) []float64 {
	// Compute group delay from unwrapped phase
	if len(frequencies) == 0 || len(phaseUnwrapped) == 0 {
		return nil
	}
	count := len(frequencies)
	if len(phaseUnwrapped) < count {
		count = len(phaseUnwrapped)
	}
	out := make([]float64, count)
	scale := -1 / (2 * math.Pi)
	for i := 0; i < count; i++ {
		// Use forward/backward/central difference
		var dPhi, dF float64
		switch {
		case i == 0 && count > 1:
			dPhi = phaseUnwrapped[i+1] - phaseUnwrapped[i]
			dF = frequencies[i+1] - frequencies[i]
		case i == count-1 && count > 1:
			dPhi = phaseUnwrapped[i] - phaseUnwrapped[i-1]
			dF = frequencies[i] - frequencies[i-1]
		case count > 2:
			dPhi = phaseUnwrapped[i+1] - phaseUnwrapped[i-1]
			dF = frequencies[i+1] - frequencies[i-1]
		default:
			// Not enough points
			out[i] = math.NaN()
			continue
		}
		if dF == 0 || math.IsNaN(dF) || math.IsNaN(dPhi) {
			// Avoid division by zero / invalid values
			out[i] = math.NaN()
			continue
		}
		delaySeconds := scale * (dPhi / dF)
		out[i] = delaySeconds * 1000
	}
	return out
}
