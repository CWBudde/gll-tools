package clf

import "math"

// GLL symmetry type constants.
const (
	SymmetryNone       int32 = 0
	SymmetryVertical   int32 = 1
	SymmetryHorizontal int32 = 2
	SymmetryQuarter    int32 = 3
	SymmetryAxial      int32 = 4
)

// GLLSymmetryToCLF maps a GLL symmetry type to the corresponding CLF symmetry tag.
func GLLSymmetryToCLF(gllSym int32) string {
	switch gllSym {
	case SymmetryNone:
		return "<none>"
	case SymmetryVertical:
		return "<vertical>"
	case SymmetryHorizontal:
		return "<horizontal>"
	case SymmetryQuarter:
		return "<full>"
	case SymmetryAxial:
		return "<rotational>"
	default:
		return "<none>"
	}
}

// AzimuthCount returns the number of azimuth angles for a given GLL
// symmetry type and angular step size in degrees.
func AzimuthCount(gllSym int32, stepDeg float64) int {
	switch gllSym {
	case SymmetryNone:
		return int(math.Round(360 / stepDeg))
	case SymmetryVertical, SymmetryHorizontal:
		return int(math.Round(180/stepDeg)) + 1
	case SymmetryQuarter:
		return int(math.Round(90/stepDeg)) + 1
	case SymmetryAxial:
		return 1
	default:
		return int(math.Round(360 / stepDeg))
	}
}
