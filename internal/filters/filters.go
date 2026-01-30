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

type filterResponseState struct {
	baseFrequencies   []float64
	levels            []float64
	phase             []float64
	hasPhase          bool
	usedFilters       int
	skippedFilters    int
	mismatchedFilters int
	totalGain         float64
	filterKind        string
	firSampleRate     float64
	firIsComplex      bool
}

func (s *filterResponseState) collectFilters(filters []gll.GenericBaseFilter) ([]genericFilterCandidate, []genericIIRCandidate) {
	var firCandidates []genericFilterCandidate
	var iirCandidates []genericIIRCandidate

	for _, filter := range filters {
		if filter.ByPass {
			continue
		}
		if filter.Kind == gll.FilterKindLogSpectrum && filter.LogSpectrum != nil {
			s.processLogSpectrum(filter)
		} else {
			switch {
			case filter.Kind == gll.FilterKindFIR && filter.FIRData != nil:
				firCandidates = append(firCandidates, genericFilterCandidate{
					Gain: filter.Gain,
					FIR:  filter.FIRData,
				})
			case filter.Kind == gll.FilterKindIIR && filter.IIRParams != nil:
				iirCandidates = append(iirCandidates, genericIIRCandidate{
					Gain:   filter.Gain,
					Params: filter.IIRParams,
				})
			default:
				s.skippedFilters++
			}
		}
	}
	return firCandidates, iirCandidates
}

func (s *filterResponseState) processLogSpectrum(filter gll.GenericBaseFilter) {
	spectrum := filter.LogSpectrum
	frequencies := BuildLogSpectrumFrequencies(spectrum)
	if len(frequencies) == 0 || len(spectrum.Level) == 0 {
		s.skippedFilters++
		return
	}

	if s.baseFrequencies == nil {
		s.baseFrequencies = frequencies
		s.levels = make([]float64, len(frequencies))
		s.phase = make([]float64, len(frequencies))
	} else if !frequenciesMatch(s.baseFrequencies, frequencies) {
		s.mismatchedFilters++
		return
	}

	if len(spectrum.Level) == len(s.levels) {
		for i, value := range spectrum.Level {
			s.levels[i] += value
		}
	}

	if len(spectrum.Phase) == len(s.phase) {
		for i, value := range spectrum.Phase {
			s.phase[i] += value
		}
		s.hasPhase = true
	}

	s.totalGain += filter.Gain
	s.usedFilters++
}

func (s *filterResponseState) processFIRCandidates(candidates []genericFilterCandidate) {
	s.filterKind = "FIR"
	for _, candidate := range candidates {
		frequencies, levelsData, phaseData, ok := buildFIRResponse(candidate.FIR)
		if !ok {
			s.skippedFilters++
			continue
		}

		if s.baseFrequencies == nil {
			s.baseFrequencies = frequencies
			s.levels = make([]float64, len(frequencies))
			s.phase = make([]float64, len(frequencies))
			s.firSampleRate = candidate.FIR.SampleRate
			s.firIsComplex = candidate.FIR.IsComplex
		} else if !frequenciesMatch(s.baseFrequencies, frequencies) {
			s.mismatchedFilters++
			continue
		}

		for i, value := range levelsData {
			s.levels[i] += value
		}
		if len(phaseData) == len(s.phase) {
			for i, value := range phaseData {
				s.phase[i] += value
			}
			s.hasPhase = true
		}

		s.totalGain += candidate.Gain
		s.usedFilters++
	}
}

func (s *filterResponseState) processIIRCandidates(candidates []genericIIRCandidate) {
	s.filterKind = "IIR"
	s.baseFrequencies = buildStandardLogSpectrumFrequencies()
	if len(s.baseFrequencies) > 0 {
		s.levels = make([]float64, len(s.baseFrequencies))
		s.phase = make([]float64, len(s.baseFrequencies))
		for _, candidate := range candidates {
			levelsData, phaseData, ok := buildIIRResponse(candidate.Params, s.baseFrequencies)
			if !ok {
				s.skippedFilters++
				continue
			}
			for i, value := range levelsData {
				s.levels[i] += value
			}
			if len(phaseData) == len(s.phase) {
				for i, value := range phaseData {
					s.phase[i] += value
				}
				s.hasPhase = true
			}
			s.totalGain += candidate.Gain
			s.usedFilters++
		}
	}
}

func (s *filterResponseState) finalize(bank *gll.GenericFilterBank) {
	if bank.ByPass {
		for i := range s.levels {
			s.levels[i] = 0
		}
		if s.hasPhase {
			for i := range s.phase {
				s.phase[i] = 0
			}
		}
	} else if s.totalGain != 0 {
		for i := range s.levels {
			s.levels[i] += s.totalGain
		}
	}
}

func (s *filterResponseState) toResult(bank *gll.GenericFilterBank) FilterResponseResult {
	message := ""
	if s.skippedFilters > 0 || s.mismatchedFilters > 0 {
		parts := make([]string, 0, 2)
		if s.skippedFilters > 0 {
			parts = append(parts, fmt.Sprintf("%d unsupported", s.skippedFilters))
		}
		if s.mismatchedFilters > 0 {
			parts = append(parts, fmt.Sprintf("%d mismatched grid", s.mismatchedFilters))
		}
		message = "Skipped " + strings.Join(parts, ", ")
	}

	result := FilterResponseResult{
		Success:           true,
		Message:           message,
		FilterKind:        s.filterKind,
		Frequencies:       s.baseFrequencies,
		Level:             s.levels,
		UsedFilters:       s.usedFilters,
		SkippedFilters:    s.skippedFilters,
		MismatchedFilters: s.mismatchedFilters,
		Bypassed:          bank.ByPass,
		SampleRate:        s.firSampleRate,
		PointCount:        len(s.levels),
		IsComplex:         s.firIsComplex,
	}
	if s.hasPhase {
		result.Phase = s.phase
	}
	return result
}

// BuildFilterResponse calculates the combined response for a filter definition.
func BuildFilterResponse(file *gll.File, req FilterResponseRequest) FilterResponseResult {
	if file.Database == nil {
		return FilterResponseResult{Success: false, Error: "no database available in GLL file"}
	}

	if req.GroupIndex < 0 || req.GroupIndex >= len(file.Database.FilterGroups) {
		return FilterResponseResult{Success: false, Error: "filter group index out of range"}
	}

	group := file.Database.FilterGroups[req.GroupIndex]
	if req.FilterIndex < 0 || req.FilterIndex >= len(group.Filters) {
		return FilterResponseResult{Success: false, Error: "filter index out of range"}
	}

	filterDef := group.Filters[req.FilterIndex]
	bank := filterDef.Filter
	if bank == nil {
		return FilterResponseResult{Success: true, Message: "No filter response data available"}
	}

	state := &filterResponseState{
		filterKind: "LogSpectrum",
		totalGain:  bank.Gain,
	}
	firCandidates, iirCandidates := state.collectFilters(bank.Filters)

	if state.baseFrequencies == nil {
		state.processFIRCandidates(firCandidates)
	}

	if state.baseFrequencies == nil && len(iirCandidates) > 0 {
		state.processIIRCandidates(iirCandidates)
	}

	if state.baseFrequencies == nil || len(state.levels) == 0 {
		return FilterResponseResult{
			Success:        true,
			Message:        "No LogSpectrum, IIR, or frequency-domain FIR filters available",
			UsedFilters:    state.usedFilters,
			SkippedFilters: state.skippedFilters,
		}
	}

	state.finalize(bank)

	return state.toResult(bank)
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
			realValue := data.DataIRM[i]
			imagValue := data.DataDIP[i]
			mag := math.Hypot(realValue, imagValue)
			levels[i-startIdx] = magnitudeToDB(mag)
			phase[i-startIdx] = math.Atan2(imagValue, realValue)
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
