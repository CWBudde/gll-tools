package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/compression"
	"github.com/cwbudde/gll-tools/internal/gll"
)

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
	IsEven         bool      `json:"is_even"`          // true=even-symmetric (-> FFT optimization)
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

// parseFilterGroupBuffer parses the FilterGroups buffer
func parseFilterGroupBuffer(br *gll.ByteReader, maxOffset int64) ([]FilterGroup, error) {
	return parseBufferItems(br, maxOffset, 0, parseFilterGroup)
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
	return parseBufferItems(br, maxOffset, 0, parseFilterDefinition)
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

	// Read LogSpectrumDefinition
	spectrum := readFilterSpectrumDefinitionLP(br, endOffset)

	// Read compression type
	compressionType, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	// Validate numBands
	numBands := spectrum.NumberOfBands
	if numBands <= 0 || numBands > 10000 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	// Dispatch based on compression type
	switch compressionType {
	case 0:
		spectrum.Level, spectrum.Phase = readUncompressedResponseData(br, numBands)
	case 1:
		spectrum.Level, spectrum.Phase = readCompressedResponseData(br, numBands, endOffset)
	}

	// Read Delay
	spectrum.Delay, _ = br.ReadDouble()

	_, _ = br.Seek(endOffset, io.SeekStart)

	return spectrum, nil
}

// readFilterSpectrumDefinitionLP reads LogSpectrumDefinition fields into TransferFunctionLP.
// On error, seeks to endOffset and returns partial data (graceful degradation).
func readFilterSpectrumDefinitionLP(br *gll.ByteReader, endOffset int64) *TransferFunctionLP {
	def, err := parseLogSpectrumDefinition(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return &TransferFunctionLP{}
	}

	return &TransferFunctionLP{
		BandsPerOctave:  def.BandsPerOctave,
		LowestFrequency: def.StartFreq,
		NumberOfBands:   def.PointCount,
	}
}

// readUncompressedResponseData reads uncompressed int16 arrays for level and phase.
// Scales values: level by 0.01 (dB), phase by 0.001 (radians).
// Returns partial data on read errors (graceful degradation).
func readUncompressedResponseData(br *gll.ByteReader, numBands int32) (levels, phases []float64) {
	// Read level data
	levelCount, _ := br.ReadInt32()
	if levelCount > 0 && levelCount <= numBands {
		levels = make([]float64, levelCount)
		for i := int32(0); i < levelCount; i++ {
			val, err := br.ReadInt16()
			if err != nil {
				break
			}
			levels[i] = float64(val) * levelScaleFactor
		}
	}

	// Read phase data
	phaseCount, _ := br.ReadInt32()
	if phaseCount > 0 && phaseCount <= numBands {
		phases = make([]float64, phaseCount)
		for i := int32(0); i < phaseCount; i++ {
			val, err := br.ReadInt16()
			if err != nil {
				break
			}
			phases[i] = float64(val) * phaseScaleFactor
		}
	}

	return levels, phases
}

// readCompressedResponseData reads BitCompressed arrays for level and phase.
// Decompresses using compression.DecompressByteArray, then scales values.
// On error, seeks to endOffset and returns partial data (graceful degradation).
func readCompressedResponseData(br *gll.ByteReader, numBands int32, endOffset int64) (levels, phases []float64) {
	// Read level data
	levelCount, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, nil
	}

	levelCompressedLen, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, nil
	}

	if levelCompressedLen < 0 || int64(levelCompressedLen)*2 > int64(numBands)*8 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, nil
	}

	levelBytes := int(levelCompressedLen) * 2
	if levelBytes < 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, nil
	}

	levelCompressed, err := br.ReadBytes(levelBytes)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, nil
	}

	if levelCount > 0 && levelCount <= numBands {
		values := compression.DecompressByteArray(levelCompressed, int(levelCount), true, 8)
		levels = make([]float64, len(values))
		for i, value := range values {
			levels[i] = float64(value) * levelScaleFactor
		}
	}

	// Read phase data
	phaseCount, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return levels, nil
	}

	phaseCompressedLen, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return levels, nil
	}

	if phaseCompressedLen < 0 || int64(phaseCompressedLen)*2 > int64(numBands)*8 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return levels, nil
	}

	phaseBytes := int(phaseCompressedLen) * 2
	if phaseBytes < 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return levels, nil
	}

	phaseCompressed, err := br.ReadBytes(phaseBytes)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return levels, nil
	}

	if phaseCount > 0 && phaseCount <= numBands {
		values := compression.DecompressByteArray(phaseCompressed, int(phaseCount), true, 8)
		phases = make([]float64, len(values))
		for i, value := range values {
			phases[i] = float64(value) * phaseScaleFactor
		}
	}

	return levels, phases
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

	definition, err := parseLogSpectrumDefinition(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, err
	}

	spectrum := &TransferFunctionLP{
		BandsPerOctave:  definition.BandsPerOctave,
		LowestFrequency: definition.StartFreq,
		NumberOfBands:   definition.PointCount,
	}

	levelData, phaseData, err := parseComplexSequence(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return spectrum, nil
	}

	spectrum.Level = make([]float64, len(levelData))
	for i, value := range levelData {
		spectrum.Level[i] = float64(value) * levelScaleFactor
	}

	spectrum.Phase = make([]float64, len(phaseData))
	for i, value := range phaseData {
		spectrum.Phase[i] = float64(value) * phaseScaleFactor
	}

	spectrum.Delay, _ = br.ReadDouble()

	_, _ = br.Seek(endOffset, io.SeekStart)

	return spectrum, nil
}
