// Package gll provides GLL file parsing and array response calculations.
package gll

import (
	"math"

	"github.com/cwbudde/gll-tools/internal/acoustics"
)

// AirProperties holds air parameters for acoustic calculations
type AirProperties struct {
	Temperature float64 // Celsius
	Humidity    float64 // Relative humidity (0-1)
	Pressure    float64 // Atmospheric pressure in kPa
	Speed       float64 // Speed of sound in m/s
}

// DefaultAirProperties returns standard air conditions (20°C, 50% humidity)
func DefaultAirProperties() AirProperties {
	temp, humidity, speed := acoustics.DefaultAirProperties()
	return AirProperties{
		Temperature: temp,
		Humidity:    humidity,
		Pressure:    acoustics.ReferencePressureKPa,
		Speed:       speed,
	}
}

// GetAirLossPerMeter returns air absorption in dB/m at the given frequency
// Based on ISO 9613-1 atmospheric absorption for pure-tone sound.
func (ap AirProperties) GetAirLossPerMeter(freq float64) float64 {
	pressure := ap.Pressure
	if pressure == 0 {
		pressure = acoustics.ReferencePressureKPa
	}
	return acoustics.AirLossPerMeter(freq, ap.Temperature, ap.Humidity, pressure)
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
func (tf *TransferFunction) ToComplex() (realValues, imagValues []float64) {
	return acoustics.ToComplex(tf.Level, tf.Phase)
}

// FromComplex converts Real/Imaginary back to Level/Phase
func (tf *TransferFunction) FromComplex(realValues, imagValues []float64) {
	level, phase := acoustics.FromComplex(realValues, imagValues)
	tf.Level = level
	tf.Phase = phase
}

// Add performs coherent complex summation with another TransferFunction.
// This preserves phase relationships for accurate interference modeling.
func (tf *TransferFunction) Add(other *TransferFunction) {
	acoustics.AddComplexInPlace(tf.Level, tf.Phase, other.Level, other.Phase)
}

// Multiply applies a filter (complex multiplication).
// In Level/Phase domain: add levels, add phases.
func (tf *TransferFunction) Multiply(filter *TransferFunction) {
	acoustics.MultiplyLevelPhase(tf.Level, tf.Phase, filter.Level, filter.Phase)
}

// AddGain adds a constant gain in dB to all frequency bands
func (tf *TransferFunction) AddGain(gainDB float64) {
	acoustics.AddGain(tf.Level, gainDB)
}

// AddDelay modifies phase based on frequency to simulate time delay.
// delay is in seconds.
func (tf *TransferFunction) AddDelay(delay float64) {
	tf.Delay += delay
	freqs := make([]float64, len(tf.Phase))
	for i := range freqs {
		freqs[i] = tf.Definition.GetFrequency(i)
	}
	acoustics.AddDelay(tf.Phase, freqs, delay)
}

// ArrayElement represents a single element in a line array configuration
type ArrayElement struct {
	Position      Vector3D            // Position in meters (world coordinates)
	Angles        Vector3D            // Rotation angles in radians (H, V, R)
	Orientation   *[9]float64         // Optional row-major world-from-local rotation matrix
	Gain          float64             // Per-element gain in dB
	SourceDefs    []*SourceDefinition // Source definitions for this element
	FilterSpectra []*TransferFunction // Combined internal+external filters per source
}

// ArrayConfig represents a complete line array configuration
type ArrayConfig struct {
	Elements []ArrayElement
}

// ArrayResponseDetails contains the summed array response and each valid
// element contribution used to build it.
type ArrayResponseDetails struct {
	TransferFunction     *TransferFunction
	ElementContributions []*TransferFunction
}

// ComputeSystemResponseGrid calculates the combined array response at multiple receiver positions.
// This is more efficient than calling ComputeSystemResponseAt in a loop because the caller
// can reuse the same parsed config and loaded balloon data.
func ComputeSystemResponseGrid(
	config *ArrayConfig,
	receivers []Vector3D,
	airProps AirProperties,
	airAttenOn bool,
) []*TransferFunction {
	return ComputeSystemResponseGridWithProgress(config, receivers, airProps, airAttenOn, nil)
}

// ComputeSystemResponseGridWithProgress calculates the combined array response
// at multiple receiver positions and reports progress after receiver batches.
func ComputeSystemResponseGridWithProgress(
	config *ArrayConfig,
	receivers []Vector3D,
	airProps AirProperties,
	airAttenOn bool,
	progress func(completed, total int),
) []*TransferFunction {
	results := make([]*TransferFunction, len(receivers))
	if progress != nil {
		progress(0, len(receivers))
	}
	progressEvery := progressReportInterval(len(receivers))
	for i, recv := range receivers {
		results[i] = ComputeSystemResponseAt(config, recv, airProps, airAttenOn)
		completed := i + 1
		if progress != nil && (completed == len(receivers) || completed%progressEvery == 0) {
			progress(completed, len(receivers))
		}
	}
	return results
}

func progressReportInterval(total int) int {
	if total <= 100 {
		return 1
	}
	interval := total / 100
	if interval < 1 {
		return 1
	}
	return interval
}

// ComputeSystemResponseAt calculates the combined array response at a receiver position.
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
	if config == nil {
		return nil
	}

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

// ComputeSystemResponseDetailsAt calculates the combined response and keeps the
// per-element spectra that participate in the coherent sum.
func ComputeSystemResponseDetailsAt(
	config *ArrayConfig,
	receiver Vector3D,
	airProps AirProperties,
	airAttenOn bool,
) *ArrayResponseDetails {
	if config == nil {
		return nil
	}

	var arraySpectrum *TransferFunction
	contributions := make([]*TransferFunction, 0, len(config.Elements))

	for i := range config.Elements {
		elem := &config.Elements[i]

		response, arrivalTime := computeElementResponseAt(
			elem,
			receiver,
			airProps,
			airAttenOn,
		)

		if response == nil {
			continue
		}

		response.AddDelay(arrivalTime)
		contributions = append(contributions, response.CopyDeep())

		if arraySpectrum == nil {
			arraySpectrum = response
		} else {
			arraySpectrum.Add(response)
		}
	}

	if arraySpectrum == nil {
		return nil
	}

	return &ArrayResponseDetails{
		TransferFunction:     arraySpectrum,
		ElementContributions: contributions,
	}
}

// computeElementResponseAt calculates response for a single array element.
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
			elem.Orientation,
			receiver,
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
	distance := acoustics.Distance(receiver.X, receiver.Y, receiver.Z, elem.Position.X, elem.Position.Y, elem.Position.Z)
	arrivalTime := distance / airProps.Speed

	return boxSpectrum, arrivalTime
}

// getSourceResponseAt gets the directivity response at a given receiver position.
func getSourceResponseAt(
	srcDef *SourceDefinition,
	sourcePos Vector3D,
	sourceAngles Vector3D,
	orientation *[9]float64,
	receiver Vector3D,
) (*TransferFunction, *TransferFunction, float64) {
	if srcDef.BalloonData == nil || len(srcDef.BalloonData.Responses) == 0 {
		return nil, nil, 1.0
	}

	// Calculate vector from source to receiver
	vecX, vecY, vecZ := acoustics.Sub(receiver.X, receiver.Y, receiver.Z, sourcePos.X, sourcePos.Y, sourcePos.Z)

	// Calculate distance and propagation factor
	distance := acoustics.Length(vecX, vecY, vecZ)
	if distance < 0.01 {
		distance = 0.01 // Minimum distance
	}
	propagationFactor := 1.0 / distance

	// Convert to explicit GLL balloon coordinates.
	var meridianDeg, parallelDeg float64
	if orientation != nil {
		meridianDeg, parallelDeg = acoustics.DirectionToGLLAnglesWithMatrix(
			vecX,
			vecY,
			vecZ,
			*orientation,
		)
	} else {
		meridianDeg, parallelDeg = acoustics.DirectionToGLLAngles(
			vecX,
			vecY,
			vecZ,
			sourceAngles.X,
			sourceAngles.Y,
			sourceAngles.Z,
		)
	}

	// Get response at that angle from balloon data (with interpolation).
	response := srcDef.BalloonData.responseAtGLLAngles(meridianDeg, parallelDeg)

	// Get on-axis response for normalization.
	onAxis := srcDef.BalloonData.responseAtGLLAngles(0, 0)

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
	phiDeg = acoustics.MapMeridianBySymmetry(phiDeg, int(bd.AngularResolution.Symmetry))

	// Calculate grid indices
	merStep := bd.AngularResolution.MeridianStep
	parStep := bd.AngularResolution.ParallelStep
	merCount := acoustics.MeridianCount(bd.AngularResolution.MeridianStep, int(bd.AngularResolution.Symmetry))

	// Bilinear interpolation indices
	merIdx := phiDeg / merStep
	parIdx := (thetaDeg + 90.0) / parStep // Shift from -90..90 to 0..180

	merIdx0 := int(math.Floor(merIdx))
	merIdx1 := int(math.Ceil(merIdx)) % merCount
	parIdx0 := int(math.Floor(parIdx))
	parIdx1 := int(math.Ceil(parIdx))

	// Clamp parallel indices
	parCount := acoustics.ParallelCount(bd.AngularResolution.ParallelStep, bd.AngularResolution.FrontHalfOnly)
	parIdx0 = acoustics.ClampParallelIndex(parIdx0, parCount)
	parIdx1 = acoustics.ClampParallelIndex(parIdx1, parCount)

	// Get interpolation weights
	// Get 4 corner responses using meridian-major indexing with pole dedup.
	idx00 := acoustics.ResponseIndex(merIdx0, parIdx0, parCount, bd.AngularResolution.FrontHalfOnly)
	idx01 := acoustics.ResponseIndex(merIdx1, parIdx0, parCount, bd.AngularResolution.FrontHalfOnly)
	idx10 := acoustics.ResponseIndex(merIdx0, parIdx1, parCount, bd.AngularResolution.FrontHalfOnly)
	idx11 := acoustics.ResponseIndex(merIdx1, parIdx1, parCount, bd.AngularResolution.FrontHalfOnly)

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
		Phase:      make([]float64, len(r00.Level)),
	}

	w00, w01, w10, w11 := acoustics.BilinearWeights(merIdx, parIdx)

	for i := range result.Level {
		result.Level[i], result.Phase[i] = interpolateComplexPressureBand(i, w00, w01, w10, w11, r00, r01, r10, r11)
	}

	return result
}

func (bd *BalloonData) responseAtGLLAngles(meridianDeg, parallelDeg float64) *TransferFunction {
	if bd == nil || len(bd.Responses) == 0 {
		return nil
	}

	symmetry := int(bd.AngularResolution.Symmetry)
	parStep := bd.AngularResolution.ParallelStep
	parCount := acoustics.ParallelCount(parStep, bd.AngularResolution.FrontHalfOnly)
	merStep := bd.AngularResolution.MeridianStep
	merCount := acoustics.MeridianCount(merStep, symmetry)
	if merCount <= 0 || parCount <= 0 || merStep <= 0 || parStep <= 0 {
		return nil
	}

	meridianDeg = normalizeGLLMeridian(meridianDeg, symmetry)
	parallelDeg, ok := normalizeGLLParallel(
		parallelDeg,
		symmetry,
		parStep,
		parCount,
		bd.AngularResolution.FrontHalfOnly,
	)
	if !ok {
		return nil
	}

	merIdx := meridianDeg / merStep
	parIdx := parallelDeg / parStep

	merIdx0 := int(math.Floor(merIdx))
	merIdx1 := int(math.Ceil(merIdx))
	if symmetry == int(SymmetryNone) && merCount > 1 {
		merIdx1 %= merCount
	} else if merIdx1 >= merCount {
		merIdx1 = merCount - 1
	}
	parIdx0 := acoustics.ClampParallelIndex(int(math.Floor(parIdx)), parCount)
	parIdx1 := acoustics.ClampParallelIndex(int(math.Ceil(parIdx)), parCount)

	idx00 := acoustics.ResponseIndex(merIdx0, parIdx0, parCount, bd.AngularResolution.FrontHalfOnly)
	idx01 := acoustics.ResponseIndex(merIdx1, parIdx0, parCount, bd.AngularResolution.FrontHalfOnly)
	idx10 := acoustics.ResponseIndex(merIdx0, parIdx1, parCount, bd.AngularResolution.FrontHalfOnly)
	idx11 := acoustics.ResponseIndex(merIdx1, parIdx1, parCount, bd.AngularResolution.FrontHalfOnly)

	maxIdx := len(bd.Responses) - 1
	idx00 = clampResponseIndex(idx00, maxIdx)
	idx01 = clampResponseIndex(idx01, maxIdx)
	idx10 = clampResponseIndex(idx10, maxIdx)
	idx11 = clampResponseIndex(idx11, maxIdx)

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
		Phase:      make([]float64, len(r00.Level)),
	}

	w00, w01, w10, w11 := acoustics.BilinearWeights(merIdx, parIdx)

	for i := range result.Level {
		result.Level[i], result.Phase[i] = interpolateComplexPressureBand(i, w00, w01, w10, w11, r00, r01, r10, r11)
	}

	return result
}

func interpolateComplexPressureBand(
	band int,
	w00, w01, w10, w11 float64,
	r00, r01, r10, r11 *TransferFunction,
) (float64, float64) {
	real00, imag00 := weightedComplexPressure(w00, r00, band)
	real01, imag01 := weightedComplexPressure(w01, r01, band)
	real10, imag10 := weightedComplexPressure(w10, r10, band)
	real11, imag11 := weightedComplexPressure(w11, r11, band)

	realSum := real00 + real01 + real10 + real11
	imagSum := imag00 + imag01 + imag10 + imag11
	if math.Abs(realSum) < 1e-12 && math.Abs(imagSum) < 1e-12 {
		return -200.0, 0.0
	}

	level := 20.0 * math.Log10(math.Hypot(realSum, imagSum))
	phase := math.Atan2(imagSum, realSum)
	return level, phase
}

func weightedComplexPressure(weight float64, response *TransferFunction, band int) (float64, float64) {
	if weight == 0 || response == nil || band < 0 || band >= len(response.Level) {
		return 0, 0
	}
	phase := 0.0
	if band < len(response.Phase) {
		phase = response.Phase[band]
	}
	magnitude := weight * math.Pow(10, response.Level[band]/20.0)
	return magnitude * math.Cos(phase), magnitude * math.Sin(phase)
}

func normalizeGLLMeridian(meridianDeg float64, symmetry int) float64 {
	for meridianDeg < 0 {
		meridianDeg += 360.0
	}
	for meridianDeg >= 360.0 {
		meridianDeg -= 360.0
	}
	return acoustics.MapMeridianBySymmetry(meridianDeg, symmetry)
}

func normalizeGLLParallel(
	parallelDeg float64,
	symmetry int,
	parStep float64,
	parCount int,
	frontHalfOnly bool,
) (float64, bool) {
	if parallelDeg < 0 || parallelDeg > 180 {
		return 0, false
	}
	if frontHalfOnly && parallelDeg > 90 {
		return 0, false
	}

	measuredParallelDeg := float64(parCount-1) * parStep
	if parallelDeg <= measuredParallelDeg {
		return parallelDeg, true
	}

	canMirrorParallel := symmetry == int(SymmetryHorizontal) || symmetry == int(SymmetryQuarter)
	if !canMirrorParallel {
		return 0, false
	}

	mirrored := 180.0 - parallelDeg
	if mirrored < 0 || mirrored > measuredParallelDeg {
		return 0, false
	}

	return mirrored, true
}

func clampResponseIndex(idx, maxIdx int) int {
	if idx > maxIdx {
		return maxIdx
	}
	return idx
}
