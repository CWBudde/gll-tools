package acoustics

import "math"

// MapMeridianBySymmetry maps meridian angle based on symmetry type.
func MapMeridianBySymmetry(merDeg float64, symmetry int) float64 {
	switch symmetry {
	case 1: // SymmetryAxial
		return 0
	case 2: // SymmetryQuarter
		if merDeg >= 270 {
			return 360 - merDeg
		} else if merDeg >= 180 {
			return merDeg - 180
		} else if merDeg >= 90 {
			return 180 - merDeg
		}
		return merDeg
	case 3: // SymmetryVertical
		if merDeg >= 180 {
			return 360 - merDeg
		}
		return merDeg
	case 4: // SymmetryHorizontal
		merDeg -= 90
		if merDeg < 0 {
			return -merDeg
		} else if merDeg >= 180 {
			return 360 - merDeg
		}
		return merDeg
	default: // SymmetryNone
		return merDeg
	}
}

// MeridianCount returns the number of meridian samples for a given symmetry and step.
func MeridianCount(step float64, symmetry int) int {
	if step <= 0 {
		return 1
	}
	switch symmetry {
	case 1: // SymmetryAxial
		return 1
	case 2: // SymmetryQuarter
		return int(90/step) + 1
	case 3, 4: // SymmetryVertical, SymmetryHorizontal
		return int(180/step) + 1
	default:
		return int(360 / step)
	}
}

// ParallelCount returns the number of parallel samples for a given step/frontHalfOnly setting.
func ParallelCount(step float64, frontHalfOnly bool) int {
	if step <= 0 {
		return 1
	}
	if frontHalfOnly {
		return int(90/step) + 1
	}
	return int(180/step) + 1
}

// ResponseIndex computes the flat array index for a (merIdx, parIdx) pair.
func ResponseIndex(merIdx, parIdx, parCount int, frontHalfOnly bool) int {
	fho := frontHalfOnly
	lastParIdx := parCount - 1

	isFrontPole := parIdx == 0
	isBackPole := parIdx == lastParIdx && !fho

	// Poles are stored only at mer=0.
	if isFrontPole || isBackPole {
		return parIdx
	}

	if merIdx == 0 {
		return parIdx
	}

	// Subsequent meridians skip poles.
	skippedPerMer := 2
	if fho {
		skippedPerMer = 1
	}
	pointsPerMer := parCount - skippedPerMer

	return parCount + (merIdx-1)*pointsPerMer + (parIdx - 1)
}

// ClampParallelIndex clamps a parallel index to valid bounds.
func ClampParallelIndex(idx, parCount int) int {
	if idx < 0 {
		return 0
	}
	if idx >= parCount {
		return parCount - 1
	}
	return idx
}

// BilinearWeights returns bilinear interpolation weights for fractional indices.
func BilinearWeights(merIdx, parIdx float64) (w00, w01, w10, w11 float64) {
	merFrac := merIdx - math.Floor(merIdx)
	parFrac := parIdx - math.Floor(parIdx)
	w00 = (1.0 - merFrac) * (1.0 - parFrac)
	w01 = merFrac * (1.0 - parFrac)
	w10 = (1.0 - merFrac) * parFrac
	w11 = merFrac * parFrac
	return w00, w01, w10, w11
}
