package acoustics

import "math"

// ToComplex converts Level/Phase to real/imaginary arrays.
func ToComplex(level, phase []float64) (realValues, imagValues []float64) {
	n := len(level)
	realValues = make([]float64, n)
	imagValues = make([]float64, n)

	for i := 0; i < n; i++ {
		magnitude := math.Pow(10, level[i]/20.0)
		realValues[i] = magnitude * math.Cos(phase[i])
		imagValues[i] = magnitude * math.Sin(phase[i])
	}
	return realValues, imagValues
}

// FromComplex converts real/imaginary arrays to Level/Phase.
func FromComplex(realValues, imagValues []float64) (level, phase []float64) {
	n := len(realValues)
	level = make([]float64, n)
	phase = make([]float64, n)

	for i := 0; i < n; i++ {
		magnitude := math.Sqrt(realValues[i]*realValues[i] + imagValues[i]*imagValues[i])
		if magnitude > 0 {
			level[i] = 20.0 * math.Log10(magnitude)
		} else {
			level[i] = -200.0
		}
		phase[i] = math.Atan2(imagValues[i], realValues[i])
	}
	return level, phase
}

// AddComplexInPlace sums another Level/Phase into level/phase coherently.
func AddComplexInPlace(level, phase, otherLevel, otherPhase []float64) {
	real1, imag1 := ToComplex(level, phase)
	real2, imag2 := ToComplex(otherLevel, otherPhase)

	for i := range real1 {
		real1[i] += real2[i]
		imag1[i] += imag2[i]
	}

	newLevel, newPhase := FromComplex(real1, imag1)
	copy(level, newLevel)
	copy(phase, newPhase)
}

// MultiplyLevelPhase applies a filter in Level/Phase domain.
func MultiplyLevelPhase(level, phase, filterLevel, filterPhase []float64) {
	for i := range level {
		level[i] += filterLevel[i]
		phase[i] += filterPhase[i]
	}
}

// AddGain adds a constant gain in dB to all frequency bands.
func AddGain(level []float64, gainDB float64) {
	for i := range level {
		level[i] += gainDB
	}
}

// AddDelay shifts phase based on frequency to simulate time delay.
func AddDelay(phase []float64, freqs []float64, delay float64) {
	for i := range phase {
		phase[i] += 2.0 * math.Pi * freqs[i] * delay
	}
}
