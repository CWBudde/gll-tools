package gll

import (
	"fmt"
	"io"

	"github.com/MeKo-Christian/gll-tools/internal/gll"
)

// LimitType represents the type of mechanical/electrical limit
type LimitType int32

const (
	LimitTypeMaxCount     LimitType = 0
	LimitTypeMaxCountType LimitType = 1
	LimitTypeMaxWeightKg  LimitType = 2
	LimitTypeMaxTiltAngle LimitType = 4
	LimitTypeMinTiltAngle LimitType = 5
	LimitTypeMinCount     LimitType = 6
)

//nolint:goconst // Enum String() methods intentionally use string literals
func (t LimitType) String() string {
	switch t {
	case LimitTypeMaxCount:
		return "MaxCount"
	case LimitTypeMaxCountType:
		return "MaxCountType"
	case LimitTypeMaxWeightKg:
		return "MaxWeightKg"
	case LimitTypeMaxTiltAngle:
		return "MaxTiltAngle"
	case LimitTypeMinTiltAngle:
		return "MinTiltAngle"
	case LimitTypeMinCount:
		return "MinCount"
	default:
		return "Unknown"
	}
}

// Limit represents a mechanical or electrical limit for array configurations
type Limit struct {
	Frame      string    `json:"frame,omitempty"`    // Frame key this applies to
	Type       LimitType `json:"type"`               // Type of limit
	BoxType    string    `json:"box_type,omitempty"` // Box type key this applies to
	LimitValue float64   `json:"limit_value"`        // The limit value
}

// WarningType represents the type of configuration warning
type WarningType int32

const (
	WarningTypeMaxCount     WarningType = 0
	WarningTypeMinCount     WarningType = 1
	WarningTypeMaxWeightKg  WarningType = 2
	WarningTypeMaxTiltAngle WarningType = 3
	WarningTypeMinTiltAngle WarningType = 4
)

//nolint:goconst // Enum String() methods intentionally use string literals
func (t WarningType) String() string {
	switch t {
	case WarningTypeMaxCount:
		return "MaxCount"
	case WarningTypeMinCount:
		return "MinCount"
	case WarningTypeMaxWeightKg:
		return "MaxWeightKg"
	case WarningTypeMaxTiltAngle:
		return "MaxTiltAngle"
	case WarningTypeMinTiltAngle:
		return "MinTiltAngle"
	default:
		return "Unknown"
	}
}

// Warning represents a configuration warning message
type Warning struct {
	Frame      string      `json:"frame,omitempty"` // Frame key this applies to
	Type       WarningType `json:"type"`            // Type of warning
	Text       string      `json:"text,omitempty"`  // Warning message text
	LimitValue float64     `json:"limit_value"`     // The limit value that triggers warning
}

// FilterGroup contains DSP filter definitions
type FilterGroup struct {
	Label         string             `json:"label"`
	Key           string             `json:"key"`
	IsOverridable bool               `json:"is_overridable,omitempty"`
	Filters       []FilterDefinition `json:"filters,omitempty"`
}

// FilterDefinition represents a single filter in a filter group
type FilterDefinition struct {
	Label  string             `json:"label"`
	Key    string             `json:"key"`
	Filter *GenericFilterBank `json:"filter,omitempty"` // Filter bank (not currently parsed)
}

// FilterKind identifies the type of filter in a filter bank
type FilterKind int32

const (
	FilterKindLogSpectrum FilterKind = 0 // Frequency response curve
	FilterKindIIR         FilterKind = 1 // IIR biquad filter
	FilterKindFIR         FilterKind = 2 // FIR filter
)

// FilterType identifies the type of IIR filter
type FilterType int32

const (
	FilterTypeLowPass   FilterType = 0
	FilterTypeHighPass  FilterType = 1
	FilterTypeAllPass   FilterType = 2
	FilterTypePeak      FilterType = 3
	FilterTypePeakSym   FilterType = 4
	FilterTypeLowShelf  FilterType = 5
	FilterTypeHighShelf FilterType = 6
)

// FilterShape identifies the shape/response of an IIR filter
type FilterShape int32

const (
	FilterShapeButterworth   FilterShape = 0 // Maximally flat magnitude
	FilterShapeLinkwitzRiley FilterShape = 1 // Linkwitz-Riley (even orders)
	FilterShapeBessel        FilterShape = 2 // Maximally flat group delay
	FilterShapeSallenKey     FilterShape = 3 // 2nd-order with adjustable Q
)

// FilterAlign identifies the frequency alignment of an IIR filter
type FilterAlign int32

const (
	FilterAlignNone         FilterAlign = 0
	FilterAlignLevel3dB     FilterAlign = 1 // -3dB at critical frequency
	FilterAlignLevel6dB     FilterAlign = 2 // -6dB at critical frequency
	FilterAlignPhaseMatched FilterAlign = 3 // Phase-matched (Bessel only)
)

// GenericFilterBank is a container for a chain of DSP filters
// Binary format: block_size + vcheck(0) + sver + reserved(0) + ByPass + InvertPolarity + Gain + Delay + Filters + [MuteInput if sver>=1]
type GenericFilterBank struct {
	ByPass         bool                `json:"bypass,omitempty"`
	InvertPolarity bool                `json:"invert_polarity,omitempty"`
	MuteInput      bool                `json:"mute_input,omitempty"`
	Gain           float64             `json:"gain"`  // dB
	Delay          float64             `json:"delay"` // seconds
	Filters        []GenericBaseFilter `json:"filters,omitempty"`
}

// GenericBaseFilter is the base structure for all filter types
// Binary format: block_size + vcheck(0) + sver + reserved(0) + ByPass + InvertPolarity + Gain + Delay + Label + Key
type GenericBaseFilter struct {
	Kind           FilterKind          `json:"kind"`
	Label          string              `json:"label,omitempty"`
	Key            string              `json:"key,omitempty"`
	ByPass         bool                `json:"bypass,omitempty"`
	InvertPolarity bool                `json:"invert_polarity,omitempty"`
	Gain           float64             `json:"gain"`                   // dB
	Delay          float64             `json:"delay"`                  // seconds
	IIRParams      *IIRFilterParams    `json:"iir_params,omitempty"`   // For FilterKindIIR
	FIRData        *FIRFilterData      `json:"fir_data,omitempty"`     // For FilterKindFIR
	LogSpectrum    *TransferFunctionLP `json:"log_spectrum,omitempty"` // For FilterKindLogSpectrum
}

// IIRFilterParams contains the parameters for an IIR biquad filter
// Binary format: block_size + vcheck(0) + sver + FilterType + FilterShape + Order + FreqCritInHz + Alignment + reserved(0.0) + QFactor + ParametricGainIndB
type IIRFilterParams struct {
	FilterType         FilterType  `json:"filter_type"`
	FilterShape        FilterShape `json:"filter_shape"`
	Order              int32       `json:"order"`        // 1-8
	FreqCritInHz       float64     `json:"freq_crit_hz"` // Critical/center frequency
	Alignment          FilterAlign `json:"alignment"`
	QFactor            float64     `json:"q_factor"`           // Q (for SallenKey)
	ParametricGainIndB float64     `json:"parametric_gain_db"` // Gain for peak/shelf filters
}

// FIRFilterData contains time or frequency domain data for an FIR filter
// Binary format: block_size + vcheck(0) + sver + flags + dataIRMLen + dataIRM[] + dataDIPLen + dataDIP[] + sampleRate
type FIRFilterData struct {
	IsTimeResponse bool      `json:"is_time_response"` // true=time domain, false=frequency domain
	IsComplex      bool      `json:"is_complex"`       // true=complex data
	IsEven         bool      `json:"is_even"`          // true=even-symmetric (FFT optimization)
	DataIRM        []float64 `json:"data_irm"`         // Real part (time) or magnitude (freq)
	DataDIP        []float64 `json:"data_dip"`         // Imaginary part (time) or phase (freq)
	SampleRate     float64   `json:"sample_rate"`      // Sample rate in Hz
}

// TransferFunctionLP represents a level/phase frequency response used by LogSpectrumFilter
// This is a simplified version of the full TransferFunctionLsPs for filter storage
type TransferFunctionLP struct {
	BandsPerOctave  int32     `json:"bands_per_octave"`
	LowestFrequency float64   `json:"lowest_frequency"`
	NumberOfBands   int32     `json:"number_of_bands"`
	Level           []float64 `json:"level,omitempty"` // dB
	Phase           []float64 `json:"phase,omitempty"` // radians
	Delay           float64   `json:"delay"`           // seconds
}

// parseLimitBuffer parses the Limits buffer
func parseLimitBuffer(br *gll.ByteReader, maxOffset int64) ([]Limit, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Limits buffer has a block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading buffer block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4

	// Read version check and sub-version
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Read count
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading limit count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	limits := make([]Limit, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		limit, err := parseLimit(br)
		if err != nil {
			break
		}

		limits = append(limits, *limit)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return limits, nil
}

func parseLimit(br *gll.ByteReader) (*Limit, error) {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	// Read version check
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	limit := &Limit{}

	// Read Frame
	limit.Frame, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading frame: %w", err)
	}

	// Read Type
	typeVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading type: %w", err)
	}

	limit.Type = LimitType(typeVal)

	// Read BoxType
	limit.BoxType, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading box type: %w", err)
	}

	// Read LimitValue
	limit.LimitValue, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading limit value: %w", err)
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return limit, nil
}

// parseWarningBuffer parses the Warnings buffer
func parseWarningBuffer(br *gll.ByteReader, maxOffset int64) ([]Warning, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Warnings buffer has a block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading buffer block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4

	// Read version check and sub-version
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Read count
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading warning count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	warnings := make([]Warning, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		warning, err := parseWarning(br)
		if err != nil {
			break
		}

		warnings = append(warnings, *warning)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return warnings, nil
}

func parseWarning(br *gll.ByteReader) (*Warning, error) {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	// Read version check
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	warning := &Warning{}

	// Read Frame
	warning.Frame, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading frame: %w", err)
	}

	// Read Type
	typeVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading type: %w", err)
	}

	warning.Type = WarningType(typeVal)

	// Read Text
	warning.Text, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading text: %w", err)
	}

	// Read LimitValue
	warning.LimitValue, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading limit value: %w", err)
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return warning, nil
}

// parseFilterGroupBuffer parses the FilterGroups buffer
func parseFilterGroupBuffer(br *gll.ByteReader, maxOffset int64) ([]FilterGroup, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// FilterGroups buffer has a block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading buffer block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4

	// Read version check and sub-version
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Read count
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading filter group count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	groups := make([]FilterGroup, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		group, err := parseFilterGroup(br)
		if err != nil {
			break
		}

		groups = append(groups, *group)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return groups, nil
}

func parseFilterGroup(br *gll.ByteReader) (*FilterGroup, error) {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	// Read version check
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	// Read sub-version
	subVersion, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	group := &FilterGroup{}

	// Read Label
	group.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read Key
	group.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading key: %w", err)
	}

	// Parse FilterDefinitions buffer (nested)
	filters, err := parseFilterDefinitionBuffer(br, endOffset)
	if err == nil {
		group.Filters = filters
	}

	// Read IsOverridable (sub_version >= 1)
	if subVersion >= 1 {
		flags, err := br.ReadInt16()
		if err == nil {
			group.IsOverridable = (flags & 1) != 0
		}
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return group, nil
}

func parseFilterDefinitionBuffer(br *gll.ByteReader, maxOffset int64) ([]FilterDefinition, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// FilterDefinitions buffer has a block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading buffer block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4
	if bufferEnd > maxOffset {
		bufferEnd = maxOffset
	}

	// Read version check and sub-version
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Read count
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading filter definition count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	filters := make([]FilterDefinition, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		filter, err := parseFilterDefinition(br)
		if err != nil {
			break
		}

		filters = append(filters, *filter)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return filters, nil
}

func parseFilterDefinition(br *gll.ByteReader) (*FilterDefinition, error) {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	// Read version check
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	filter := &FilterDefinition{}

	// Read Label
	filter.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read Key
	filter.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading key: %w", err)
	}

	// Parse GenericFilterBank
	filterBank, err := parseGenericFilterBank(br, endOffset)
	if err == nil {
		filter.Filter = filterBank
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return filter, nil
}

// parseGenericFilterBank parses a GenericFilterBank structure
func parseGenericFilterBank(br *gll.ByteReader, maxOffset int64) (*GenericFilterBank, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	subVersion, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	bank := &GenericFilterBank{}

	// Read reserved field (always 0)
	_, _ = br.ReadInt32()

	// Read ByPass
	bypassVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return bank, nil
	}
	bank.ByPass = bypassVal != 0

	// Read InvertPolarity
	invertVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return bank, nil
	}
	bank.InvertPolarity = invertVal != 0

	// Read Gain
	bank.Gain, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return bank, nil
	}

	// Read Delay
	bank.Delay, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return bank, nil
	}

	// Parse GenericBaseFilterBuffer
	filters, err := parseGenericBaseFilterBuffer(br, endOffset)
	if err == nil {
		bank.Filters = filters
	}

	// Read MuteInput (sub_version >= 1)
	if subVersion >= 1 {
		muteVal, err := br.ReadInt32()
		if err == nil {
			bank.MuteInput = muteVal != 0
		}
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return bank, nil
}

// parseGenericBaseFilterBuffer parses a buffer of GenericBaseFilter items
func parseGenericBaseFilterBuffer(br *gll.ByteReader, maxOffset int64) ([]GenericBaseFilter, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading buffer block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	bufferEnd := br.Offset() + int64(blockSize) - 4
	if bufferEnd > maxOffset {
		bufferEnd = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, fmt.Errorf("unsupported buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading filter count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	filters := make([]GenericBaseFilter, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		// Read FilterKind discriminator
		kindVal, err := br.ReadInt32()
		if err != nil {
			break
		}

		kind := FilterKind(kindVal)

		var filter *GenericBaseFilter
		switch kind {
		case FilterKindLogSpectrum:
			filter, err = parseLogSpectrumFilter(br, bufferEnd)
		case FilterKindIIR:
			filter, err = parseIIRFilter(br, bufferEnd)
		case FilterKindFIR:
			filter, err = parseFIRFilter(br, bufferEnd)
		default:
			// Unknown filter kind, skip to end
			break
		}

		if err != nil || filter == nil {
			break
		}

		filter.Kind = kind
		filters = append(filters, *filter)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return filters, nil
}

// parseGenericBaseFilterBase parses the common base fields of all filter types
// Note: The base fields have their own inner block wrapper
func parseGenericBaseFilterBase(br *gll.ByteReader, maxOffset int64) (*GenericBaseFilter, error) {
	filter := &GenericBaseFilter{}

	if br.Offset() >= maxOffset {
		return filter, nil
	}

	// Read inner block size for base fields
	blockSize, err := br.ReadInt32()
	if err != nil {
		return filter, err
	}

	if blockSize <= 0 {
		return filter, nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	// Read version check (must be 0)
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return filter, err
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return filter, err
	}

	// Read reserved field (always 0)
	_, _ = br.ReadInt32()

	// Read ByPass
	bypassVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}
	filter.ByPass = bypassVal != 0

	// Read InvertPolarity
	invertVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}
	filter.InvertPolarity = invertVal != 0

	// Read Gain
	filter.Gain, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}

	// Read Delay
	filter.Delay, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}

	// Read Label
	filter.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}

	// Read Key
	filter.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return filter, nil
}

// parseLogSpectrumFilter parses a LogSpectrumFilter (FilterKind=0)
func parseLogSpectrumFilter(br *gll.ByteReader, maxOffset int64) (*GenericBaseFilter, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	// LogSpectrumFilter accepts vcheck=0 or vcheck=1
	if versionCheck < 0 || versionCheck > 1 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Parse base fields
	filter, err := parseGenericBaseFilterBase(br, endOffset)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}

	// Parse spectrum data based on vcheck
	if versionCheck == 0 {
		// Older CLogSpectrumLP format
		spectrum, err := parseFilterLogSpectrumLP(br, endOffset)
		if err == nil {
			filter.LogSpectrum = spectrum
		}
	} else {
		// TransferFunctionLsPs format
		spectrum, err := parseFilterTransferFunctionLP(br, endOffset)
		if err == nil {
			filter.LogSpectrum = spectrum
		}
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return filter, nil
}

// parseIIRFilter parses an IIRFilter (FilterKind=1)
func parseIIRFilter(br *gll.ByteReader, maxOffset int64) (*GenericBaseFilter, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Parse base fields
	filter, err := parseGenericBaseFilterBase(br, endOffset)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}

	// Parse FilterFunction
	params, err := parseFilterFunction(br, endOffset)
	if err == nil {
		filter.IIRParams = params
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return filter, nil
}

// parseFIRFilter parses a FIRFilter (FilterKind=2)
func parseFIRFilter(br *gll.ByteReader, maxOffset int64) (*GenericBaseFilter, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	// Parse base fields
	filter, err := parseGenericBaseFilterBase(br, endOffset)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return filter, nil
	}

	// Parse CLinResponse
	firData, err := parseCLinResponse(br, endOffset)
	if err == nil {
		filter.FIRData = firData
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return filter, nil
}

// parseFilterFunction parses the IIR filter parameters
func parseFilterFunction(br *gll.ByteReader, maxOffset int64) (*IIRFilterParams, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	params := &IIRFilterParams{}

	// Read FilterType
	typeVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return params, nil
	}
	params.FilterType = FilterType(typeVal)

	// Read FilterShape
	shapeVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return params, nil
	}
	params.FilterShape = FilterShape(shapeVal)

	// Read Order
	params.Order, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return params, nil
	}

	// Read FreqCritInHz
	params.FreqCritInHz, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return params, nil
	}

	// Read Alignment
	alignVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return params, nil
	}
	params.Alignment = FilterAlign(alignVal)

	// Read reserved (always 0.0)
	_, _ = br.ReadDouble()

	// Read QFactor
	params.QFactor, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return params, nil
	}

	// Read ParametricGainIndB
	params.ParametricGainIndB, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return params, nil
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return params, nil
}

// parseCLinResponse parses FIR filter time/frequency domain data
func parseCLinResponse(br *gll.ByteReader, maxOffset int64) (*FIRFilterData, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	data := &FIRFilterData{}

	// Read flags
	flags, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return data, nil
	}
	data.IsTimeResponse = (flags & 1) != 0
	data.IsComplex = (flags & 2) != 0
	data.IsEven = (flags & 4) != 0

	// Read dataIRM array
	irmLen, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return data, nil
	}

	if irmLen > 0 && irmLen < 1000000 { // Sanity check
		data.DataIRM = make([]float64, irmLen)
		for i := int32(0); i < irmLen; i++ {
			data.DataIRM[i], err = br.ReadDouble()
			if err != nil {
				break
			}
		}
	}

	// Read dataDIP array
	dipLen, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return data, nil
	}

	if dipLen > 0 && dipLen < 1000000 { // Sanity check
		data.DataDIP = make([]float64, dipLen)
		for i := int32(0); i < dipLen; i++ {
			data.DataDIP[i], err = br.ReadDouble()
			if err != nil {
				break
			}
		}
	}

	// Read SampleRate
	data.SampleRate, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return data, nil
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return data, nil
}

// parseFilterLogSpectrumLP parses the older CLogSpectrumLP format (vcheck=0)
func parseFilterLogSpectrumLP(br *gll.ByteReader, maxOffset int64) (*TransferFunctionLP, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	// CLogSpectrumLP can have vcheck=0 or vcheck=1
	if versionCheck < 0 || versionCheck > 1 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	spectrum := &TransferFunctionLP{}

	// Read LogSpectrumDefinition
	spectrum.BandsPerOctave, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	spectrum.LowestFrequency, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	spectrum.NumberOfBands, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	// Read compression type
	compressionType, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	numBands := spectrum.NumberOfBands
	if numBands <= 0 || numBands > 10000 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	if compressionType == 0 {
		// Uncompressed: int16 arrays
		levelCount, _ := br.ReadInt32()
		if levelCount > 0 && levelCount <= numBands {
			spectrum.Level = make([]float64, levelCount)
			for i := int32(0); i < levelCount; i++ {
				val, err := br.ReadInt16()
				if err != nil {
					break
				}
				spectrum.Level[i] = float64(val) * 0.01 // Scale to dB
			}
		}

		phaseCount, _ := br.ReadInt32()
		if phaseCount > 0 && phaseCount <= numBands {
			spectrum.Phase = make([]float64, phaseCount)
			for i := int32(0); i < phaseCount; i++ {
				val, err := br.ReadInt16()
				if err != nil {
					break
				}
				spectrum.Phase[i] = float64(val) * 0.001 // Scale to radians
			}
		}
	}
	// Compressed format (compressionType=1) uses BitCompression - skip for now

	// Read Delay
	spectrum.Delay, _ = br.ReadDouble()

	_, _ = br.Seek(endOffset, io.SeekStart)

	return spectrum, nil
}

// parseFilterTransferFunctionLP parses the TransferFunctionLsPs format (vcheck=1)
func parseFilterTransferFunctionLP(br *gll.ByteReader, maxOffset int64) (*TransferFunctionLP, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil
	}

	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	spectrum := &TransferFunctionLP{}

	// Read LogSpectrumDefinition
	spectrum.BandsPerOctave, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	spectrum.LowestFrequency, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	spectrum.NumberOfBands, err = br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	// Skip ComplexSequence block - it uses the same Record format as balloon data
	// For simplicity, we skip the raw data and just get the definition
	// Full parsing would require reusing the bitcompression code from source.go

	_, _ = br.Seek(endOffset, io.SeekStart)

	return spectrum, nil
}
