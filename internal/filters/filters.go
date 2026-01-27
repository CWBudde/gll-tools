package filters

import (
	"fmt"
	"math"
	"strings"

	"github.com/cwbudde/gll-tools/pkg/gll"
)

// FilterResponseRequest selects a filter definition by index.
type FilterResponseRequest struct {
	GroupIndex  int `json:"group_index"`
	FilterIndex int `json:"filter_index"`
}

// FilterResponseResult is the output of a filter response computation.
type FilterResponseResult struct {
	Success           bool      `json:"success"`
	Error             string    `json:"error,omitempty"`
	Message           string    `json:"message,omitempty"`
	FilterKind        string    `json:"filter_kind,omitempty"`
	Frequencies       []float64 `json:"frequencies,omitempty"`
	Level             []float64 `json:"level,omitempty"`
	Phase             []float64 `json:"phase,omitempty"`
	UsedFilters       int       `json:"used_filters,omitempty"`
	SkippedFilters    int       `json:"skipped_filters,omitempty"`
	MismatchedFilters int       `json:"mismatched_filters,omitempty"`
	Bypassed          bool      `json:"bypassed,omitempty"`
	SampleRate        float64   `json:"sample_rate,omitempty"`
	PointCount        int       `json:"point_count,omitempty"`
	IsComplex         bool      `json:"is_complex,omitempty"`
}

// BuildFilterResponse calculates the combined response for a filter definition.
func BuildFilterResponse(file *gll.File, req FilterResponseRequest) FilterResponseResult {
	if file.Database == nil {
		return FilterResponseResult{
			Success: false,
			Error:   "no database available in GLL file",
		}
	}

	if req.GroupIndex < 0 || req.GroupIndex >= len(file.Database.FilterGroups) {
		return FilterResponseResult{
			Success: false,
			Error:   "filter group index out of range",
		}
	}

	group := file.Database.FilterGroups[req.GroupIndex]
	if req.FilterIndex < 0 || req.FilterIndex >= len(group.Filters) {
		return FilterResponseResult{
			Success: false,
			Error:   "filter index out of range",
		}
	}

	filterDef := group.Filters[req.FilterIndex]
	bank := filterDef.Filter
	if bank == nil {
		return FilterResponseResult{
			Success: true,
			Message: "No filter response data available",
		}
	}

	var baseFrequencies []float64
	var levels []float64
	var phase []float64
	hasPhase := false
	usedFilters := 0
	skippedFilters := 0
	mismatchedFilters := 0
	totalGain := bank.Gain
	filterKind := "LogSpectrum"
	firCandidates := make([]genericFilterCandidate, 0, len(bank.Filters))
	iirCandidates := make([]genericIIRCandidate, 0, len(bank.Filters))

	for _, filter := range bank.Filters {
		if filter.ByPass {
			continue
		}
		if filter.Kind != gll.FilterKindLogSpectrum || filter.LogSpectrum == nil {
			if filter.Kind == gll.FilterKindFIR && filter.FIRData != nil {
				firCandidates = append(firCandidates, genericFilterCandidate{
					Gain: filter.Gain,
					FIR:  filter.FIRData,
				})
			} else if filter.Kind == gll.FilterKindIIR && filter.IIRParams != nil {
				iirCandidates = append(iirCandidates, genericIIRCandidate{
					Gain:   filter.Gain,
					Params: filter.IIRParams,
				})
			} else {
				skippedFilters++
			}
			continue
		}

		spectrum := filter.LogSpectrum
		frequencies := BuildLogSpectrumFrequencies(spectrum)
		if len(frequencies) == 0 || len(spectrum.Level) == 0 {
			skippedFilters++
			continue
		}

		if baseFrequencies == nil {
			baseFrequencies = frequencies
			levels = make([]float64, len(frequencies))
			phase = make([]float64, len(frequencies))
		} else if !frequenciesMatch(baseFrequencies, frequencies) {
			mismatchedFilters++
			continue
		}

		if len(spectrum.Level) == len(levels) {
			for i, value := range spectrum.Level {
				levels[i] += value
			}
		}

		if len(spectrum.Phase) == len(phase) {
			for i, value := range spectrum.Phase {
				phase[i] += value
			}
			hasPhase = true
		}

		totalGain += filter.Gain
		usedFilters++
	}

	var firSampleRate float64
	var firIsComplex bool

	if baseFrequencies == nil {
		filterKind = "FIR"
		for _, candidate := range firCandidates {
			frequencies, levelsData, phaseData, ok := buildFIRResponse(candidate.FIR)
			if !ok {
				skippedFilters++
				continue
			}

			if baseFrequencies == nil {
				baseFrequencies = frequencies
				levels = make([]float64, len(frequencies))
				phase = make([]float64, len(frequencies))
				firSampleRate = candidate.FIR.SampleRate
				firIsComplex = candidate.FIR.IsComplex
			} else if !frequenciesMatch(baseFrequencies, frequencies) {
				mismatchedFilters++
				continue
			}

			for i, value := range levelsData {
				levels[i] += value
			}
			if len(phaseData) == len(phase) {
				for i, value := range phaseData {
					phase[i] += value
				}
				hasPhase = true
			}

			totalGain += candidate.Gain
			usedFilters++
		}
	}

	if baseFrequencies == nil && len(iirCandidates) > 0 {
		filterKind = "IIR"
		baseFrequencies = buildStandardLogSpectrumFrequencies()
		if len(baseFrequencies) > 0 {
			levels = make([]float64, len(baseFrequencies))
			phase = make([]float64, len(baseFrequencies))
			for _, candidate := range iirCandidates {
				levelsData, phaseData, ok := buildIIRResponse(candidate.Params, baseFrequencies)
				if !ok {
					skippedFilters++
					continue
				}
				for i, value := range levelsData {
					levels[i] += value
				}
				if len(phaseData) == len(phase) {
					for i, value := range phaseData {
						phase[i] += value
					}
					hasPhase = true
				}
				totalGain += candidate.Gain
				usedFilters++
			}
		}
	}

	if baseFrequencies == nil || len(levels) == 0 {
		return FilterResponseResult{
			Success:        true,
			Message:        "No LogSpectrum, IIR, or frequency-domain FIR filters available",
			UsedFilters:    usedFilters,
			SkippedFilters: skippedFilters,
		}
	}

	if bank.ByPass {
		for i := range levels {
			levels[i] = 0
		}
		if hasPhase {
			for i := range phase {
				phase[i] = 0
			}
		}
	} else if totalGain != 0 {
		for i := range levels {
			levels[i] += totalGain
		}
	}

	message := ""
	if skippedFilters > 0 || mismatchedFilters > 0 {
		parts := make([]string, 0, 2)
		if skippedFilters > 0 {
			parts = append(parts, fmt.Sprintf("%d unsupported", skippedFilters))
		}
		if mismatchedFilters > 0 {
			parts = append(parts, fmt.Sprintf("%d mismatched grid", mismatchedFilters))
		}
		message = "Skipped " + strings.Join(parts, ", ")
	}

	result := FilterResponseResult{
		Success:           true,
		Message:           message,
		FilterKind:        filterKind,
		Frequencies:       baseFrequencies,
		Level:             levels,
		UsedFilters:       usedFilters,
		SkippedFilters:    skippedFilters,
		MismatchedFilters: mismatchedFilters,
		Bypassed:          bank.ByPass,
		SampleRate:        firSampleRate,
		PointCount:        len(levels),
		IsComplex:         firIsComplex,
	}
	if hasPhase {
		result.Phase = phase
	}
	return result
}

// BuildLogSpectrumFrequencies builds the frequency axis for a stored LogSpectrum response.
func BuildLogSpectrumFrequencies(spectrum *gll.TransferFunctionLP) []float64 {
	if spectrum == nil {
		return nil
	}
	if spectrum.BandsPerOctave == 0 || spectrum.LowestFrequency == 0 {
		return nil
	}
	count := len(spectrum.Level)
	if count == 0 {
		count = len(spectrum.Phase)
	}
	if count == 0 {
		count = int(spectrum.NumberOfBands)
	}
	if count == 0 {
		return nil
	}

	frequencies := make([]float64, count)
	for i := range frequencies {
		frequencies[i] = spectrum.LowestFrequency * math.Pow(2, float64(i)/float64(spectrum.BandsPerOctave))
	}
	return frequencies
}

type genericFilterCandidate struct {
	Gain float64
	FIR  *gll.FIRFilterData
}

type genericIIRCandidate struct {
	Gain   float64
	Params *gll.IIRFilterParams
}

func frequenciesMatch(a, b []float64) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	const tol = 1e-3
	for i := range a {
		av := a[i]
		bv := b[i]
		if math.IsNaN(av) || math.IsNaN(bv) || math.IsInf(av, 0) || math.IsInf(bv, 0) {
			return false
		}
		rel := math.Abs(av-bv) / math.Max(1, math.Abs(bv))
		if rel > tol {
			return false
		}
	}
	return true
}

func buildFIRResponse(data *gll.FIRFilterData) ([]float64, []float64, []float64, bool) {
	if data == nil || data.IsTimeResponse {
		return nil, nil, nil, false
	}
	if data.SampleRate <= 0 {
		return nil, nil, nil, false
	}
	count := len(data.DataIRM)
	if count < 2 {
		return nil, nil, nil, false
	}

	frequencies := buildFIRFrequencies(data.SampleRate, count)
	if len(frequencies) == 0 {
		return nil, nil, nil, false
	}

	startIdx := 0
	if frequencies[0] <= 0 {
		startIdx = 1
	}
	if startIdx >= count {
		return nil, nil, nil, false
	}

	points := count - startIdx
	levels := make([]float64, points)
	var phase []float64
	hasPhase := false

	if data.IsComplex {
		if len(data.DataDIP) != count {
			return nil, nil, nil, false
		}
		phase = make([]float64, points)
		for i := startIdx; i < count; i++ {
			real := data.DataIRM[i]
			imag := data.DataDIP[i]
			mag := math.Hypot(real, imag)
			levels[i-startIdx] = magnitudeToDB(mag)
			phase[i-startIdx] = math.Atan2(imag, real)
		}
		hasPhase = true
	} else {
		for i := startIdx; i < count; i++ {
			levels[i-startIdx] = magnitudeToDB(data.DataIRM[i])
		}
		if len(data.DataDIP) == count {
			phase = make([]float64, points)
			copy(phase, data.DataDIP[startIdx:])
			hasPhase = true
		}
	}

	return frequencies[startIdx:], levels, phase, hasPhase
}

func buildFIRFrequencies(sampleRate float64, count int) []float64 {
	if sampleRate <= 0 || count < 2 {
		return nil
	}
	nyquist := sampleRate * 0.5
	step := nyquist / float64(count-1)
	frequencies := make([]float64, count)
	for i := 0; i < count; i++ {
		frequencies[i] = step * float64(i)
	}
	return frequencies
}

func magnitudeToDB(value float64) float64 {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return -120
	}
	db := 20 * math.Log10(value)
	if db < -120 {
		return -120
	}
	return db
}

func buildStandardLogSpectrumFrequencies() []float64 {
	const pointCount = 241
	const bandsPerOctave = 24.0
	const bandsBelow1k = 132.0
	startFreq := 1000.0 * math.Pow(2, -bandsBelow1k/bandsPerOctave)
	frequencies := make([]float64, pointCount)
	for i := range pointCount {
		frequencies[i] = startFreq * math.Pow(2, float64(i)/bandsPerOctave)
	}
	return frequencies
}
