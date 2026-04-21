package acoustics

import "math"

// Sub returns the vector difference a-b.
func Sub(ax, ay, az, bx, by, bz float64) (dx, dy, dz float64) {
	return ax - bx, ay - by, az - bz
}

// Length returns the Euclidean length of a vector.
func Length(x, y, z float64) float64 {
	return math.Sqrt(x*x + y*y + z*z)
}

// Distance returns the Euclidean distance between two points.
func Distance(ax, ay, az, bx, by, bz float64) float64 {
	dx, dy, dz := Sub(ax, ay, az, bx, by, bz)
	return Length(dx, dy, dz)
}

// Rotate rotates vector (x,y,z) by rx, ry, rz (radians) around X,Y,Z axes.
func Rotate(x, y, z, rx, ry, rz float64) (nx, ny, nz float64) {
	// Rotation around Z axis (azimuth)
	cosZ, sinZ := math.Cos(rz), math.Sin(rz)
	x1 := x*cosZ - y*sinZ
	y1 := x*sinZ + y*cosZ
	z1 := z

	// Rotation around Y axis (elevation)
	cosY, sinY := math.Cos(ry), math.Sin(ry)
	x2 := x1*cosY + z1*sinY
	y2 := y1
	z2 := -x1*sinY + z1*cosY

	// Rotation around X axis (roll)
	cosX, sinX := math.Cos(rx), math.Sin(rx)
	x3 := x2
	y3 := y2*cosX - z2*sinX
	z3 := y2*sinX + z2*cosX

	return x3, y3, z3
}

// ThetaPhi converts a direction vector to spherical angles relative to source orientation.
func ThetaPhi(vecX, vecY, vecZ, angX, angY, angZ float64) (theta, phi float64) {
	// Rotate vector by inverse of source angles
	rx, ry, rz := Rotate(vecX, vecY, vecZ, -angX, -angY, -angZ)

	// Convert to spherical coordinates
	r := Length(rx, ry, rz)
	if r < 1e-10 {
		return 0, 0
	}

	// theta (elevation): angle from XY plane
	theta = math.Asin(rz / r)

	// phi (azimuth): angle in XY plane from X axis
	phi = math.Atan2(ry, rx)

	return theta, phi
}

// DirectionToGLLAngles converts a direction vector to GLL balloon coordinates.
// Meridian rotates around the firing axis: 0°=top (+Z), 90°=right (+Y).
// Parallel is the angle from the firing axis: 0°=front (+X), 180°=back (-X).
func DirectionToGLLAngles(vecX, vecY, vecZ, angX, angY, angZ float64) (meridianDeg, parallelDeg float64) {
	// Rotate vector by inverse of source angles
	rx, ry, rz := Rotate(vecX, vecY, vecZ, -angX, -angY, -angZ)

	r := Length(rx, ry, rz)
	if r < 1e-10 {
		return 0, 0
	}

	cosParallel := rx / r
	if cosParallel > 1 {
		cosParallel = 1
	} else if cosParallel < -1 {
		cosParallel = -1
	}
	parallelDeg = math.Acos(cosParallel) * 180.0 / math.Pi

	if math.Abs(ry) < 1e-10 && math.Abs(rz) < 1e-10 {
		return 0, parallelDeg
	}

	meridianDeg = math.Atan2(ry, rz) * 180.0 / math.Pi
	if meridianDeg < 0 {
		meridianDeg += 360.0
	}

	return meridianDeg, parallelDeg
}
