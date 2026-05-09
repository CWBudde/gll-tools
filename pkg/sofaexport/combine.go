package sofaexport

import (
	"fmt"
	"math"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// complexFromLevelPhase converts a (Level dB, Phase rad) pair to a (real,
// imag) pair of the underlying complex transfer function.
func complexFromLevelPhase(levelDB, phaseRad float64) (re, im float64) {
	a := math.Pow(10, levelDB/20.0)
	return a * math.Cos(phaseRad), a * math.Sin(phaseRad)
}

// combineResponse converts a single balloon response to (real, imag) slices,
// optionally combined with the source's OnAxisSpectrum and OnAxisLevel gain.
//
// When relative is true the response is taken as-is. When false (the default)
// the response is multiplied by srcDef.OnAxisSpectrum (pointwise complex
// multiply, same frequency grid required) and scaled by 10^(OnAxisLevel/20).
func combineResponse(resp *gll.TransferFunction, srcDef *gll.SourceDefinition, relative bool) (reArr, imArr []float64, err error) {
	if resp == nil {
		return nil, nil, fmt.Errorf("nil response")
	}
	if len(resp.Level) != len(resp.Phase) {
		return nil, nil, fmt.Errorf("response Level/Phase length mismatch (%d vs %d)", len(resp.Level), len(resp.Phase))
	}

	n := len(resp.Level)
	reArr = make([]float64, n)
	imArr = make([]float64, n)

	if relative || srcDef == nil || srcDef.OnAxisSpectrum == nil {
		for i := range n {
			reArr[i], imArr[i] = complexFromLevelPhase(resp.Level[i], resp.Phase[i])
		}
		return reArr, imArr, nil
	}

	onAxis := srcDef.OnAxisSpectrum
	if len(onAxis.Level) != n {
		return nil, nil, fmt.Errorf("OnAxisSpectrum length %d does not match response length %d", len(onAxis.Level), n)
	}
	if onAxis.Definition != resp.Definition {
		return nil, nil, fmt.Errorf("OnAxisSpectrum frequency grid (%+v) differs from response grid (%+v)",
			onAxis.Definition, resp.Definition)
	}

	gain := math.Pow(10, srcDef.OnAxisLevel/20.0)
	for i := range n {
		// Combined level/phase: add levels, add phases (complex multiplication).
		level := resp.Level[i] + onAxis.Level[i]
		phase := resp.Phase[i] + onAxis.Phase[i]
		r, im := complexFromLevelPhase(level, phase)
		reArr[i] = gain * r
		imArr[i] = gain * im
	}
	return reArr, imArr, nil
}
