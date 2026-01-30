package viz

import (
	"errors"
	"math"

	"github.com/cwbudde/gll-tools/internal/acoustics"
	"github.com/cwbudde/gll-tools/pkg/gll"
)

type PolarSlices struct {
	AnglesDeg       []float64
	HorizontalLevel []float64
	VerticalLevel   []float64
	FrequencyHz     float64
	UsesOnAxis      bool
	StepDeg         float64
	Symmetry        int32
}

type balloonGrid struct {
	meridianCount     int
	parallelCount     int
	measuredMeridian  float64
	measuredParallel  float64
	symmetry          int32
	frontHalfOnly     bool
	meridianStepDeg   float64
	parallelStepDeg   float64
	canMirrorParallel bool
}

func BuildFrequencyList(def gll.LogSpectrumDefinition) []float64 {
	// Build frequency list from spectrum definition
	if def.PointCount <= 0 || def.BandsPerOctave <= 0 || def.StartFreq <= 0 {
		return nil
	}
	freqs := make([]float64, def.PointCount)
	for i := 0; i < int(def.PointCount); i++ {
		freqs[i] = def.GetFrequency(i)
	}
	return freqs
}

func FindNearestFrequencyIndex(freqs []float64, targetHz float64) int {
	// Return index of closest frequency to target
	if len(freqs) == 0 {
		return -1
	}
	if targetHz <= 0 {
		return 0
	}
	bestIdx := 0
	bestDelta := math.Abs(freqs[0] - targetHz)
	for i := 1; i < len(freqs); i++ {
		delta := math.Abs(freqs[i] - targetHz)
		if delta < bestDelta {
			bestDelta = delta
			bestIdx = i
		}
	}
	return bestIdx
}

func ComputePolarSlices(def *gll.SourceDefinition, freqIndex int, stepDeg float64, combineOnAxis bool) (*PolarSlices, error) {
	// Validate input and balloon data
	if def == nil || def.BalloonData == nil {
		return nil, errors.New("no balloon data")
	}
	bd := def.BalloonData
	if len(bd.Responses) == 0 {
		return nil, errors.New("balloon responses not loaded")
	}
	if stepDeg <= 0 {
		stepDeg = 10
	}

	// Build grid metadata from balloon data
	grid := buildBalloonGrid(bd)
	if grid == nil {
		return nil, errors.New("invalid balloon grid")
	}

	// Create angle list and result buffers
	angles := buildPolarAngles(stepDeg)
	horizontal := make([]float64, 0, len(angles))
	vertical := make([]float64, 0, len(angles))

	// Determine whether on-axis data can be combined
	usesOnAxis := false
	var onAxisLevels []float64
	if combineOnAxis && def.OnAxisSpectrum != nil && len(def.OnAxisSpectrum.Level) > 0 {
		if sameSpectrumGrid(def.OnAxisSpectrum.Definition, bd.Responses[0].Definition) {
			usesOnAxis = true
			onAxisLevels = def.OnAxisSpectrum.Level
		}
	}

	for _, angle := range angles {
		// Horizontal slice: meridian 90 (right) for positive angles,
		// meridian 270 (left) for negative angles.
		hParallel := math.Abs(angle)
		hMeridian := 90.0
		if angle < 0 {
			hMeridian = 270.0
		}
		hResp := responseWithSymmetry(bd, grid, hMeridian, hParallel)
		hVal := math.NaN()
		if hResp != nil && freqIndex >= 0 && freqIndex < len(hResp.Level) {
			hVal = hResp.Level[freqIndex]
			if usesOnAxis && freqIndex < len(onAxisLevels) {
				hVal += onAxisLevels[freqIndex]
			}
		}
		horizontal = append(horizontal, hVal)

		// Vertical slice: meridian 0 (top) for positive angles,
		// meridian 180 (bottom) for negative angles.
		vParallel := math.Abs(angle)
		vMeridian := 0.0
		if angle < 0 {
			vMeridian = 180.0
		}
		vResp := responseWithSymmetry(bd, grid, vMeridian, vParallel)
		vVal := math.NaN()
		if vResp != nil && freqIndex >= 0 && freqIndex < len(vResp.Level) {
			vVal = vResp.Level[freqIndex]
			if usesOnAxis && freqIndex < len(onAxisLevels) {
				vVal += onAxisLevels[freqIndex]
			}
		}
		vertical = append(vertical, vVal)
	}

	// Resolve frequency at index for metadata
	freq := 0.0
	if len(bd.Responses) > 0 && len(bd.Responses[0].Level) > 0 {
		defn := bd.Responses[0].Definition
		freqs := BuildFrequencyList(defn)
		if freqIndex >= 0 && freqIndex < len(freqs) {
			freq = freqs[freqIndex]
		}
	}

	return &PolarSlices{
		AnglesDeg:       angles,
		HorizontalLevel: horizontal,
		VerticalLevel:   vertical,
		FrequencyHz:     freq,
		UsesOnAxis:      usesOnAxis,
		StepDeg:         stepDeg,
		Symmetry:        bd.AngularResolution.Symmetry,
	}, nil
}

func buildPolarAngles(stepDeg float64) []float64 {
	// Build sweep of angles from 0 to ±180
	angles := []float64{0}
	for angle := -stepDeg; angle >= -180; angle -= stepDeg {
		angles = append(angles, angle)
	}
	for angle := 180 - stepDeg; angle > 0; angle -= stepDeg {
		angles = append(angles, angle)
	}
	return angles
}

func sameSpectrumGrid(a, b gll.LogSpectrumDefinition) bool {
	// Compare spectrum definitions for grid compatibility
	return a.BandsPerOctave == b.BandsPerOctave && a.StartFreq == b.StartFreq && a.PointCount == b.PointCount
}

func buildBalloonGrid(bd *gll.BalloonData) *balloonGrid {
	// Derive balloon grid dimensions from metadata
	if bd == nil {
		return nil
	}
	ang := bd.AngularResolution
	if ang.MeridianStep <= 0 || ang.ParallelStep <= 0 {
		return nil
	}

	// Read symmetry flags
	symmetry := ang.Symmetry
	frontHalfOnly := ang.FrontHalfOnly

	fullMeridianCount := int(math.Round(360.0 / ang.MeridianStep))
	if fullMeridianCount < 1 {
		fullMeridianCount = 1
	}
	fullParallelCount := int(math.Round(180.0/ang.ParallelStep)) + 1
	if fullParallelCount < 1 {
		fullParallelCount = 1
	}

	// Apply symmetry and half-sphere constraints
	meridianCount := meridianCountForSymmetry(ang.MeridianStep, symmetry, fullMeridianCount)
	parallelCount := parallelCountForHalf(ang.ParallelStep, frontHalfOnly, fullParallelCount)

	// Validate against response count (accounting for pole de-duplication)
	responseCount := len(bd.Responses)
	expectedWithDedup := meridianCount * parallelCount
	if meridianCount > 1 {
		expectedWithDedup -= meridianCount - 1
		if !frontHalfOnly {
			expectedWithDedup -= meridianCount - 1
		}
	}
	if responseCount > 0 && responseCount != meridianCount*parallelCount && responseCount != expectedWithDedup {
		// Fallback: derive parallelCount from responseCount
		altParallel := int(math.Ceil(float64(responseCount) / float64(meridianCount)))
		if altParallel > 0 && altParallel <= fullParallelCount {
			parallelCount = altParallel
		}
	}

	// Measured angular ranges (based on stored grid)
	measuredMeridian := float64(meridianCount-1) * ang.MeridianStep
	measuredParallel := float64(parallelCount-1) * ang.ParallelStep

	canMirrorParallel := symmetry == int32(gll.SymmetryHorizontal) || symmetry == int32(gll.SymmetryQuarter)

	return &balloonGrid{
		meridianCount:     meridianCount,
		parallelCount:     parallelCount,
		measuredMeridian:  measuredMeridian,
		measuredParallel:  measuredParallel,
		symmetry:          symmetry,
		frontHalfOnly:     frontHalfOnly,
		meridianStepDeg:   ang.MeridianStep,
		parallelStepDeg:   ang.ParallelStep,
		canMirrorParallel: canMirrorParallel,
	}
}

func meridianCountForSymmetry(step float64, symmetry int32, fullMeridianCount int) int {
	// Compute meridian count based on symmetry mode
	if step <= 0 {
		return 1
	}
	switch symmetry {
	case int32(gll.SymmetryAxial):
		return 1
	case int32(gll.SymmetryQuarter):
		return int(math.Round(90.0/step)) + 1
	case int32(gll.SymmetryVertical), int32(gll.SymmetryHorizontal):
		return int(math.Round(180.0/step)) + 1
	default:
		if fullMeridianCount > 0 {
			return fullMeridianCount
		}
		return int(math.Round(360.0 / step))
	}
}

func parallelCountForHalf(step float64, frontHalfOnly bool, fullParallelCount int) int {
	// Compute parallel count based on front-half setting
	if step <= 0 {
		return 1
	}
	if frontHalfOnly {
		return int(math.Round(90.0/step)) + 1
	}
	if fullParallelCount > 0 {
		return fullParallelCount
	}
	return int(math.Round(180.0/step)) + 1
}

func ResponseAtAngles(bd *gll.BalloonData, meridianDeg, parallelDeg float64) *gll.TransferFunction {
	// Public helper to query response at angles
	if bd == nil {
		return nil
	}
	grid := buildBalloonGrid(bd)
	if grid == nil {
		return nil
	}
	return responseWithSymmetry(bd, grid, meridianDeg, parallelDeg)
}

func responseWithSymmetry(bd *gll.BalloonData, grid *balloonGrid, meridianDeg, parallelDeg float64) *gll.TransferFunction {
	// Apply symmetry rules and map angles to response index
	if bd == nil || grid == nil || len(bd.Responses) == 0 {
		return nil
	}

	lookupMeridian := math.Mod(meridianDeg, 360)
	if lookupMeridian < 0 {
		lookupMeridian += 360
	}
	lookupParallel := parallelDeg

	// Fold angles based on symmetry
	switch grid.symmetry {
	case int32(gll.SymmetryAxial):
		lookupMeridian = 0
	case int32(gll.SymmetryQuarter):
		switch {
		case lookupMeridian >= 270:
			lookupMeridian = 360 - lookupMeridian
		case lookupMeridian >= 180:
			lookupMeridian -= 180
		case lookupMeridian >= 90:
			lookupMeridian = 180 - lookupMeridian
		}
	case int32(gll.SymmetryVertical):
		if lookupMeridian >= 180 {
			lookupMeridian = 360 - lookupMeridian
		}
	case int32(gll.SymmetryHorizontal):
		lookupMeridian -= 90
		switch {
		case lookupMeridian < 0:
			lookupMeridian = -lookupMeridian
		case lookupMeridian >= 180:
			lookupMeridian = 360 - lookupMeridian
		}
	}

	if lookupParallel < 0 || lookupParallel > 180 {
		return nil
	}
	if grid.frontHalfOnly && lookupParallel > 90 {
		return nil
	}
	if lookupParallel > grid.measuredParallel {
		// Mirror parallel if symmetry allows
		if grid.canMirrorParallel {
			mirrored := 180 - lookupParallel
			if mirrored <= grid.measuredParallel {
				lookupParallel = mirrored
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	meridianIdx := int(math.Round(lookupMeridian / grid.meridianStepDeg))
	parallelIdx := int(math.Round(lookupParallel / grid.parallelStepDeg))
	if meridianIdx < 0 || meridianIdx >= grid.meridianCount {
		return nil
	}
	if parallelIdx < 0 || parallelIdx >= grid.parallelCount {
		return nil
	}

	responseIndex := acoustics.ResponseIndex(meridianIdx, parallelIdx, grid.parallelCount, grid.frontHalfOnly)
	if responseIndex < 0 || responseIndex >= len(bd.Responses) {
		return nil
	}

	return &bd.Responses[responseIndex]
}
