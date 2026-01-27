// Package gll provides GLL file parsing and array response calculations.
//
// This file implements the array response calculation algorithm as reverse-engineered
// from the EASE GLL Viewer. See docs/response.md Section 10 for detailed documentation.
package gll

import (
	"math"
)

// AirProperties holds air parameters for acoustic calculations
type AirProperties struct {
	Temperature float64 // Celsius
	Humidity    float64 // Relative humidity (0-1)
	Speed       float64 // Speed of sound in m/s
}

// DefaultAirProperties returns standard air conditions (20°C, 50% humidity)
func DefaultAirProperties() AirProperties {
	return AirProperties{
		Temperature: 20.0,
		Humidity:    0.5,
		Speed:       343.0, // m/s at 20°C
	}
}

// GetAirLossPerMeter returns air absorption in dB/m at the given frequency
// Based on simplified ISO 9613-1 model
func (ap AirProperties) GetAirLossPerMeter(freq float64) float64 {
	// Simplified air absorption model
	// Real implementation would use full ISO 9613-1 equations
	return 0.001 * math.Pow(freq/1000.0, 1.5) * (1.0 - ap.Humidity*0.5)
}

// CopyDeep creates a deep copy of the TransferFunction
func (tf *TransferFunction) CopyDeep() *TransferFunction {
	result := &TransferFunction{
		Definition: tf.Definition,
		Delay:      tf.Delay,
		Level:      make([]float64, len(tf.Level)),
		Phase:      make([]float64, len(tf.Phase)),
	}
	copy(result.Level, tf.Level)
	copy(result.Phase, tf.Phase)
	return result
}

// ToComplex converts Level/Phase representation to Real/Imaginary
func (tf *TransferFunction) ToComplex() (real, imag []float64) {
	n := len(tf.Level)
	real = make([]float64, n)
	imag = make([]float64, n)

	for i := 0; i < n; i++ {
		// Convert dB to linear magnitude
		magnitude := math.Pow(10, tf.Level[i]/20.0)
		// Convert to complex components
		real[i] = magnitude * math.Cos(tf.Phase[i])
		imag[i] = magnitude * math.Sin(tf.Phase[i])
	}
	return real, imag
}

// FromComplex converts Real/Imaginary back to Level/Phase
func (tf *TransferFunction) FromComplex(real, imag []float64) {
	n := len(real)
	tf.Level = make([]float64, n)
	tf.Phase = make([]float64, n)

	for i := 0; i < n; i++ {
		magnitude := math.Sqrt(real[i]*real[i] + imag[i]*imag[i])
		if magnitude > 0 {
			tf.Level[i] = 20.0 * math.Log10(magnitude)
		} else {
			tf.Level[i] = -200.0 // Very small value for zero magnitude
		}
		tf.Phase[i] = math.Atan2(imag[i], real[i])
	}
}

// Add performs coherent complex summation with another TransferFunction.
// This preserves phase relationships for accurate interference modeling.
func (tf *TransferFunction) Add(other *TransferFunction) {
	// Convert both to complex representation
	real1, imag1 := tf.ToComplex()
	real2, imag2 := other.ToComplex()

	// Sum in complex domain
	for i := range real1 {
		real1[i] += real2[i]
		imag1[i] += imag2[i]
	}

	// Convert back to Level/Phase
	tf.FromComplex(real1, imag1)
}

// Multiply applies a filter (complex multiplication).
// In Level/Phase domain: add levels, add phases.
func (tf *TransferFunction) Multiply(filter *TransferFunction) {
	for i := range tf.Level {
		tf.Level[i] += filter.Level[i]
		tf.Phase[i] += filter.Phase[i]
	}
}

// AddGain adds a constant gain in dB to all frequency bands
func (tf *TransferFunction) AddGain(gainDB float64) {
	for i := range tf.Level {
		tf.Level[i] += gainDB
	}
}

// AddDelay modifies phase based on frequency to simulate time delay.
// delay is in seconds.
func (tf *TransferFunction) AddDelay(delay float64) {
	tf.Delay += delay
	for i := range tf.Phase {
		freq := tf.Definition.GetFrequency(i)
		// phase += 2π × frequency × delay
		tf.Phase[i] += 2.0 * math.Pi * freq * delay
	}
}

// ArrayElement represents a single element in a line array configuration
type ArrayElement struct {
	Position      Vector3D            // Position in meters (world coordinates)
	Angles        Vector3D            // Rotation angles in radians (H, V, R)
	Gain          float64             // Per-element gain in dB
	SourceDefs    []*SourceDefinition // Source definitions for this element
	FilterSpectra []*TransferFunction // Combined internal+external filters per source
}

// ArrayConfig represents a complete line array configuration
type ArrayConfig struct {
	Elements []ArrayElement
}

// ComputeSystemResponseAt calculates the combined array response at a receiver position.
// This is the Go equivalent of EASE's GenSystemInstance.ComputeSystemResponseAt.
//
// Algorithm:
//  1. For each element in the array:
//     a. Compute the element's response at the receiver position
//     b. Add arrival time delay for time-alignment
//  2. Coherently sum all element responses (preserving phase)
func ComputeSystemResponseAt(
	config *ArrayConfig,
	receiver Vector3D,
	airProps AirProperties,
	airAttenOn bool,
) *TransferFunction {
	var arraySpectrum *TransferFunction

	for i := range config.Elements {
		elem := &config.Elements[i]

		// Compute response for this element
		response, arrivalTime := computeElementResponseAt(
			elem,
			receiver,
			airProps,
			airAttenOn,
		)

		if response == nil {
			continue
		}

		// Add arrival time delay (time-align to acoustic center)
		response.AddDelay(arrivalTime)

		if arraySpectrum == nil {
			arraySpectrum = response
		} else {
			// Complex (coherent) sum
			arraySpectrum.Add(response)
		}
	}

	return arraySpectrum
}

// computeElementResponseAt calculates response for a single array element.
// This corresponds to EASE's BoxType.GetResponseAt.
func computeElementResponseAt(
	elem *ArrayElement,
	receiver Vector3D,
	airProps AirProperties,
	airAttenOn bool,
) (*TransferFunction, float64) {
	if len(elem.SourceDefs) == 0 {
		return nil, 0
	}

	var boxSpectrum *TransferFunction

	for sourceIdx, srcDef := range elem.SourceDefs {
		if srcDef == nil || srcDef.BalloonData == nil {
			continue
		}

		// Get directivity response at receiver angle
		response, onAxis, propagationFactor := getSourceResponseAt(
			srcDef,
			elem.Position,
			elem.Angles,
			receiver,
			airProps,
		)

		if response == nil {
			continue
		}

		// Get combined filter spectrum (internal + external + gain)
		if sourceIdx < len(elem.FilterSpectra) && elem.FilterSpectra[sourceIdx] != nil {
			// Apply filter (complex multiply)
			response.Multiply(elem.FilterSpectra[sourceIdx])
		}

		// Apply element gain
		response.AddGain(elem.Gain)

		// Apply on-axis normalization
		if onAxis != nil {
			response.Multiply(onAxis)
		}

		// Apply distance attenuation (1/r spherical spreading)
		response.AddGain(20.0 * math.Log10(propagationFactor))

		// Apply air absorption if enabled
		if airAttenOn {
			distance := 1.0 / propagationFactor
			for band := 0; band < len(response.Level); band++ {
				freq := response.Definition.GetFrequency(band)
				response.Level[band] -= distance * airProps.GetAirLossPerMeter(freq)
			}
		}

		// Add to box spectrum (coherent sum for multi-way speakers)
		if boxSpectrum == nil {
			boxSpectrum = response
		} else {
			boxSpectrum.Add(response)
		}
	}

	// Calculate arrival time based on distance from element to receiver
	distance := vectorLength(vectorSub(receiver, elem.Position))
	arrivalTime := distance / airProps.Speed

	return boxSpectrum, arrivalTime
}

// getSourceResponseAt gets the directivity response at a given receiver position.
// This corresponds to EASE's Source.GetResponseAt.
func getSourceResponseAt(
	srcDef *SourceDefinition,
	sourcePos Vector3D,
	sourceAngles Vector3D,
	receiver Vector3D,
	airProps AirProperties,
) (*TransferFunction, *TransferFunction, float64) {
	if srcDef.BalloonData == nil || len(srcDef.BalloonData.Responses) == 0 {
		return nil, nil, 1.0
	}

	// Calculate vector from source to receiver
	vec := vectorSub(receiver, sourcePos)

	// Calculate distance and propagation factor
	distance := vectorLength(vec)
	if distance < 0.01 {
		distance = 0.01 // Minimum distance
	}
	propagationFactor := 1.0 / distance

	// Convert to spherical angles (theta = elevation, phi = azimuth)
	theta, phi := getThetaPhi(vec, sourceAngles)

	// Get response at that angle from balloon data (with interpolation)
	response := srcDef.BalloonData.GetResponseAtAngle(theta, phi)

	// Get on-axis response for normalization
	onAxis := srcDef.BalloonData.GetResponseAtAngle(0, 0)

	if response != nil {
		// Add propagation delay
		response.AddDelay(distance / airProps.Speed)
	}

	return response, onAxis, propagationFactor
}

// GetResponseAtAngle retrieves the interpolated response at the given angles.
// theta is elevation (radians), phi is azimuth (radians).
func (bd *BalloonData) GetResponseAtAngle(theta, phi float64) *TransferFunction {
	if bd == nil || len(bd.Responses) == 0 {
		return nil
	}

	// Convert angles to degrees for grid lookup
	thetaDeg := theta * 180.0 / math.Pi
	phiDeg := phi * 180.0 / math.Pi

	// Normalize angles to grid range
	for phiDeg < 0 {
		phiDeg += 360.0
	}
	for phiDeg >= 360.0 {
		phiDeg -= 360.0
	}

	// Apply symmetry mapping
	phiDeg = bd.mapMeridianBySymmetry(phiDeg)

	// Calculate grid indices
	merStep := bd.AngularResolution.MeridianStep
	parStep := bd.AngularResolution.ParallelStep
	merCount := bd.getMeridianCount()

	// Bilinear interpolation indices
	merIdx := phiDeg / merStep
	parIdx := (thetaDeg + 90.0) / parStep // Shift from -90..90 to 0..180

	merIdx0 := int(math.Floor(merIdx))
	merIdx1 := int(math.Ceil(merIdx)) % merCount
	parIdx0 := int(math.Floor(parIdx))
	parIdx1 := int(math.Ceil(parIdx))

	// Clamp parallel indices
	parCount := bd.getParallelCount()
	if parIdx0 >= parCount {
		parIdx0 = parCount - 1
	}
	if parIdx1 >= parCount {
		parIdx1 = parCount - 1
	}

	// Get interpolation weights
	merFrac := merIdx - float64(merIdx0)
	parFrac := parIdx - float64(parIdx0)

	// Get 4 corner responses
	idx00 := parIdx0*merCount + merIdx0
	idx01 := parIdx0*merCount + merIdx1
	idx10 := parIdx1*merCount + merIdx0
	idx11 := parIdx1*merCount + merIdx1

	// Clamp indices to available responses
	maxIdx := len(bd.Responses) - 1
	if idx00 > maxIdx {
		idx00 = maxIdx
	}
	if idx01 > maxIdx {
		idx01 = maxIdx
	}
	if idx10 > maxIdx {
		idx10 = maxIdx
	}
	if idx11 > maxIdx {
		idx11 = maxIdx
	}

	// Bilinear interpolation - get references to corner responses
	r00 := &bd.Responses[idx00]
	r01 := &bd.Responses[idx01]
	r10 := &bd.Responses[idx10]
	r11 := &bd.Responses[idx11]

	if len(r00.Level) == 0 {
		return nil
	}

	result := &TransferFunction{
		Definition: r00.Definition,
		Level:      make([]float64, len(r00.Level)),
		Phase:      make([]float64, len(r00.Phase)),
	}

	w00 := (1.0 - merFrac) * (1.0 - parFrac)
	w01 := merFrac * (1.0 - parFrac)
	w10 := (1.0 - merFrac) * parFrac
	w11 := merFrac * parFrac

	for i := range result.Level {
		level00, level01, level10, level11 := 0.0, 0.0, 0.0, 0.0
		phase00, phase01, phase10, phase11 := 0.0, 0.0, 0.0, 0.0

		if i < len(r00.Level) {
			level00, phase00 = r00.Level[i], r00.Phase[i]
		}
		if i < len(r01.Level) {
			level01, phase01 = r01.Level[i], r01.Phase[i]
		}
		if i < len(r10.Level) {
			level10, phase10 = r10.Level[i], r10.Phase[i]
		}
		if i < len(r11.Level) {
			level11, phase11 = r11.Level[i], r11.Phase[i]
		}

		result.Level[i] = w00*level00 + w01*level01 + w10*level10 + w11*level11
		result.Phase[i] = w00*phase00 + w01*phase01 + w10*phase10 + w11*phase11
	}

	return result
}

// mapMeridianBySymmetry maps meridian angle based on balloon symmetry type
func (bd *BalloonData) mapMeridianBySymmetry(merDeg float64) float64 {
	switch bd.AngularResolution.Symmetry {
	case 0: // Axial - all meridians map to 0
		return 0
	case 1: // Quarter - fold 360° into 0-90°
		if merDeg >= 270 {
			return 360 - merDeg
		} else if merDeg >= 180 {
			return merDeg - 180
		} else if merDeg >= 90 {
			return 180 - merDeg
		}
		return merDeg
	case 2: // Vertical - fold 180-360° into 0-180°
		if merDeg >= 180 {
			return 360 - merDeg
		}
		return merDeg
	case 3: // Horizontal - offset by 90°, then mirror
		merDeg -= 90
		if merDeg < 0 {
			return -merDeg
		} else if merDeg >= 180 {
			return 360 - merDeg
		}
		return merDeg
	default: // None - no mapping
		return merDeg
	}
}

func (bd *BalloonData) getMeridianCount() int {
	step := bd.AngularResolution.MeridianStep
	if step <= 0 {
		return 1
	}
	switch bd.AngularResolution.Symmetry {
	case 0: // Axial
		return 1
	case 1: // Quarter
		return int(90/step) + 1
	case 2, 3: // Vertical, Horizontal
		return int(180/step) + 1
	default: // None
		return int(360 / step)
	}
}

func (bd *BalloonData) getParallelCount() int {
	step := bd.AngularResolution.ParallelStep
	if step <= 0 {
		return 1
	}
	if bd.AngularResolution.FrontHalfOnly {
		return int(90/step) + 1
	}
	return int(180/step) + 1
}

// getThetaPhi converts a direction vector to spherical angles relative to source orientation
func getThetaPhi(vec Vector3D, sourceAngles Vector3D) (theta, phi float64) {
	// Rotate vector by inverse of source angles
	rotated := rotateVector(vec, -sourceAngles.X, -sourceAngles.Y, -sourceAngles.Z)

	// Convert to spherical coordinates
	r := vectorLength(rotated)
	if r < 1e-10 {
		return 0, 0
	}

	// theta (elevation): angle from XY plane
	theta = math.Asin(rotated.Z / r)

	// phi (azimuth): angle in XY plane from X axis
	phi = math.Atan2(rotated.Y, rotated.X)

	return theta, phi
}

// Helper functions for vector math

func vectorSub(a, b Vector3D) Vector3D {
	return Vector3D{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

func vectorLength(v Vector3D) float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func rotateVector(v Vector3D, rx, ry, rz float64) Vector3D {
	// Rotation around Z axis (azimuth)
	cosZ, sinZ := math.Cos(rz), math.Sin(rz)
	x1 := v.X*cosZ - v.Y*sinZ
	y1 := v.X*sinZ + v.Y*cosZ
	z1 := v.Z

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

	return Vector3D{X: x3, Y: y3, Z: z3}
}
