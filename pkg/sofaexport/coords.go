package sofaexport

import (
	"math"

	sofa "github.com/cwbudde/go-sofa"
)

// gridAngles returns the (azimuthDeg, elevationDeg) pair for a given
// (merIdx, parIdx) pair in the GLL angular grid.
//
// GLL convention (verified against pkg/gll/array_calculations.go
// GetResponseAtAngle): meridian phi sweeps 0..360 along the equator,
// parallel parIdx maps to elevation theta = parIdx*parStep - 90, so
// parIdx=0 corresponds to theta=-90° (south pole, -Z) and the largest
// parIdx corresponds to theta=+90° (north pole, +Z). On-axis (+X) is
// (mer=0, theta=0).
func gridAngles(merIdx, parIdx int, merStep, parStep float64) (azDeg, elDeg float64) {
	azDeg = float64(merIdx) * merStep
	elDeg = float64(parIdx)*parStep - 90.0
	return azDeg, elDeg
}

// directionToCartesian converts an (azimuth, elevation, radius) triple in
// degrees/metres to a SOFA Vector3 (right-handed cartesian, meters).
func directionToCartesian(azDeg, elDeg, radius float64) sofa.Vector3 {
	az := azDeg * math.Pi / 180.0
	el := elDeg * math.Pi / 180.0
	cosEl := math.Cos(el)
	return sofa.Vector3{
		X: radius * cosEl * math.Cos(az),
		Y: radius * cosEl * math.Sin(az),
		Z: radius * math.Sin(el),
	}
}
