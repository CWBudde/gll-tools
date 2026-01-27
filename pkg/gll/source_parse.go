package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// DataType represents the frequency resolution type
type DataType int32

const (
	DataTypeHighRes     DataType = 0 // High resolution (EQTones)
	DataTypeThirdOctave DataType = 1 // 1/3 octave bands
	DataTypeOctave      DataType = 2 // Octave bands
)

//nolint:goconst // Enum String() methods intentionally use string literals
func (d DataType) String() string {
	switch d {
	case DataTypeHighRes:
		return "HighRes"
	case DataTypeThirdOctave:
		return "1/3 Octave"
	case DataTypeOctave:
		return "Octave"
	default:
		return "Unknown"
	}
}

// SymmetryType represents the directivity pattern symmetry.
// Note: on-disk codes are mapped to this enum in parseResolutionDescriptor.
type SymmetryType int32

const (
	SymmetryNone       SymmetryType = 0 // No symmetry (full 3D balloon)
	SymmetryVertical   SymmetryType = 1 // Vertical plane symmetry
	SymmetryHorizontal SymmetryType = 2 // Horizontal plane symmetry
	SymmetryQuarter    SymmetryType = 3 // Quarter symmetry
	SymmetryAxial      SymmetryType = 4 // Axial (rotational) symmetry
)

//nolint:goconst // Enum String() methods intentionally use string literals
func (s SymmetryType) String() string {
	switch s {
	case SymmetryNone:
		return "None"
	case SymmetryVertical:
		return "Vertical"
	case SymmetryHorizontal:
		return "Horizontal"
	case SymmetryQuarter:
		return "Quarter"
	case SymmetryAxial:
		return "Axial"
	default:
		return "Unknown"
	}
}

// DirectivityType represents the far-field directivity model
type DirectivityType int32

const (
	DirectivityPoint             DirectivityType = 0
	DirectivityLine              DirectivityType = 1
	DirectivityCircularPiston    DirectivityType = 2
	DirectivityRectangularPiston DirectivityType = 3
)

//nolint:goconst // Enum String() methods intentionally use string literals
func (d DirectivityType) String() string {
	switch d {
	case DirectivityPoint:
		return "Point"
	case DirectivityLine:
		return "Line"
	case DirectivityCircularPiston:
		return "CircularPiston"
	case DirectivityRectangularPiston:
		return "RectangularPiston"
	default:
		return "Unknown"
	}
}

// ResolutionDescriptor defines the angular grid for balloon data
type ResolutionDescriptor struct {
	Symmetry      int32   `json:"symmetry"`        // Balloon symmetry type
	FrontHalfOnly bool    `json:"front_half_only"` // Only front hemisphere
	MeridianStep  float64 `json:"meridian_step"`   // Horizontal angle step (degrees)
	ParallelStep  float64 `json:"parallel_step"`   // Vertical angle step (degrees)
}

// MeridianCount returns the number of horizontal measurement points
func (r ResolutionDescriptor) MeridianCount() int {
	if r.MeridianStep <= 0 {
		return 0
	}

	return int(360.0 / r.MeridianStep)
}

// ParallelCount returns the number of vertical measurement points
func (r ResolutionDescriptor) ParallelCount() int {
	if r.ParallelStep <= 0 {
		return 0
	}

	return int(180.0/r.ParallelStep) + 1
}

// TotalPoints returns total measurement points in the balloon
func (r ResolutionDescriptor) TotalPoints() int {
	return r.MeridianCount() * r.ParallelCount()
}

// BalloonData contains directivity measurements at all angles
type BalloonData struct {
	Flags             int32                `json:"flags"`
	AngularResolution ResolutionDescriptor `json:"angular_resolution"`
	ResponseCount     int32                `json:"response_count"`      // Number of responses in the file
	ResponseVersion   int16                `json:"response_version"`    // 0 = legacy CLogSpectrumLP, 1 = TransferFunctionLsPs
	ResponsesOffset   int64                `json:"-"`                   // Offset in file where responses start
	Responses         []TransferFunction   `json:"responses,omitempty"` // One per angle point
}

// SourceDefinition contains complete acoustic data for a driver/source
type SourceDefinition struct {
	// Header
	Label                string   `json:"label"`
	CompanyLabel         string   `json:"company_label,omitempty"`
	Description          string   `json:"description,omitempty"`
	NominalBandwidthFrom float64  `json:"nominal_bandwidth_from"` // Hz
	NominalBandwidthTo   float64  `json:"nominal_bandwidth_to"`   // Hz
	DataType             DataType `json:"data_type"`

	// Directivity
	BalloonData *BalloonData `json:"balloon_data,omitempty"`

	// On-axis response
	OnAxisLevel    float64           `json:"on_axis_level,omitempty"` // Reference level (dB)
	OnAxisSpectrum *TransferFunction `json:"on_axis_spectrum,omitempty"`

	// Impedance
	RatedImpedance float64           `json:"rated_impedance,omitempty"` // Ohms
	Impedance      *TransferFunction `json:"impedance,omitempty"`

	// Power handling
	MaxVoltage float64 `json:"max_voltage,omitempty"` // Vrms

	// Measurement conditions
	MeasuredVoltage     float64 `json:"measured_voltage,omitempty"`  // V (default 2.828V = 1W@8ohm)
	MeasuredDistance    float64 `json:"measured_distance,omitempty"` // m (default 1m)
	MeasuredGainIndB    float64 `json:"measured_gain_in_db,omitempty"`
	Temperature         float64 `json:"temperature,omitempty"`          // °C (default 25)
	Humidity            float64 `json:"humidity,omitempty"`             // % (default 60)
	AtmosphericPressure float64 `json:"atmospheric_pressure,omitempty"` // kPa (default 101.325)

	// Coverage angles
	RatedHorizontalAngle float64 `json:"rated_horizontal_angle,omitempty"` // degrees
	RatedVerticalAngle   float64 `json:"rated_vertical_angle,omitempty"`   // degrees

	// Far-field model
	DirectivityType DirectivityType `json:"directivity_type,omitempty"`
}

// SourceDefinitionItem wraps a SourceDefinition with its key
type SourceDefinitionItem struct {
	Key        string            `json:"key"`
	Definition *SourceDefinition `json:"definition"`
}

// parseSourceDefinitionBuffer parses the source definitions buffer
func parseSourceDefinitionBuffer(br *gll.ByteReader) ([]SourceDefinitionItem, error) {
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading source count: %w", err)
	}

	if count <= 0 {
		return nil, nil
	}

	sources := make([]SourceDefinitionItem, 0, count)

	for i := range count {
		item, err := parseSourceDefinitionItem(br)
		if err != nil {
			return sources, fmt.Errorf("parsing source %d: %w", i, err)
		}

		sources = append(sources, *item)
	}

	return sources, nil
}

func parseSourceDefinitionItem(br *gll.ByteReader) (*SourceDefinitionItem, error) {
	item := &SourceDefinitionItem{}

	// Read item block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading item block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid item block size: %d", blockSize)
	}

	itemEndOffset := br.Offset() + int64(blockSize) - 4 // -4 because blockSize includes itself

	// Read version check and sub-version
	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading item version check: %w", err)
	}

	if vcheck != 0 {
		_, _ = br.Seek(itemEndOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported item version: %d", vcheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading item sub-version: %w", err)
	}

	// Read key
	key, err := br.ReadString()
	if err != nil {
		_, _ = br.Seek(itemEndOffset, io.SeekStart)
		return nil, fmt.Errorf("reading key: %w", err)
	}

	item.Key = key

	// Parse the SourceDefinition block
	def, err := parseSourceDefinition(br)
	if err != nil {
		_, _ = br.Seek(itemEndOffset, io.SeekStart)
		return nil, fmt.Errorf("parsing definition: %w", err)
	}

	item.Definition = def

	// Seek to end of item block
	_, _ = br.Seek(itemEndOffset, io.SeekStart)

	return item, nil
}

func parseSourceDefinition(br *gll.ByteReader) (*SourceDefinition, error) {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4 // -4 because blockSize includes itself

	// Read version check (valid values: 0-5)
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck < 0 || versionCheck > 5 {
		// Skip to end of block
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	// Read sub-version
	subVersion, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	def := &SourceDefinition{}

	// Read Label
	def.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read CompanyLabel
	def.CompanyLabel, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading company label: %w", err)
	}

	// Read Description
	def.Description, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading description: %w", err)
	}

	// Read NominalBandwidthFrom
	def.NominalBandwidthFrom, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading bandwidth from: %w", err)
	}

	// Read NominalBandwidthTo
	def.NominalBandwidthTo, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading bandwidth to: %w", err)
	}

	// Read DataType
	dataType, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading data type: %w", err)
	}

	def.DataType = DataType(dataType)

	// Try to parse BalloonData
	balloon, err := parseBalloonData(br, endOffset)
	if err != nil {
		// Non-fatal: skip balloon parsing and continue
		def.BalloonData = nil
	} else {
		def.BalloonData = balloon
	}

	// Parse on-axis spectrum section (best-effort)
	onAxisFlags, err := br.ReadInt32()
	if err == nil {
		if level, err := br.ReadDouble(); err == nil {
			def.OnAxisLevel = level
		}

		// Optional CLinResponse reference file (skip if present)
		if onAxisFlags&1 != 0 {
			_ = skipBlock(br, endOffset)
		}

		// OnAxisSpectrumRaw
		var onAxisTF *TransferFunction
		if versionCheck < 4 {
			onAxisTF, err = parseCLogSpectrumLP(br)
		} else {
			onAxisTF, err = parseTransferFunctionLsPs(br)
		}
		if err == nil && onAxisTF != nil {
			def.OnAxisSpectrum = onAxisTF
		}
	}

	// Skip additional fields for now - seek to end
	_, err = br.Seek(endOffset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seeking to block end: %w", err)
	}

	// Set defaults for measurement conditions
	if def.MeasuredVoltage == 0 {
		def.MeasuredVoltage = 2.828 // 1W @ 8 ohms
	}

	if def.MeasuredDistance == 0 {
		def.MeasuredDistance = 1.0
	}

	if def.Temperature == 0 {
		def.Temperature = 25.0
	}

	if def.Humidity == 0 {
		def.Humidity = 60.0
	}

	if def.AtmosphericPressure == 0 {
		def.AtmosphericPressure = 101.325
	}

	_ = subVersion // Will use for conditional parsing later

	return def, nil
}

func skipBlock(br *gll.ByteReader, maxOffset int64) error {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return err
	}
	if blockSize <= 0 {
		return nil
	}
	endOffset := br.Offset() + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}
	_, err = br.Seek(endOffset, io.SeekStart)
	return err
}

func parseBalloonData(br *gll.ByteReader, maxOffset int64) (*BalloonData, error) {
	if br.Offset() >= maxOffset {
		return nil, fmt.Errorf("past block end")
	}

	balloon := &BalloonData{}

	// Read block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading balloon block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid balloon block size: %d", blockSize)
	}

	balloonEndOffset := br.Offset() + int64(blockSize) - 4 // -4 because blockSize includes itself

	// Read version check (valid: 0-1)
	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading balloon version check: %w", err)
	}

	if vcheck < 0 || vcheck > 1 {
		_, _ = br.Seek(balloonEndOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported balloon version: %d", vcheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading balloon sub-version: %w", err)
	}

	// Read flags
	flags, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading flags: %w", err)
	}

	balloon.Flags = flags

	// Read AngularResolution (has its own block wrapper)
	res, err := parseResolutionDescriptor(br)
	if err != nil {
		return nil, fmt.Errorf("reading resolution: %w", err)
	}

	balloon.AngularResolution = *res

	// Read response count
	responseCount, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading response count: %w", err)
	}

	balloon.ResponseCount = responseCount
	balloon.ResponseVersion = vcheck
	balloon.ResponsesOffset = br.Offset() // Save offset for lazy loading

	// For large datasets, we don't load all responses into memory by default
	// They can be loaded using LoadBalloonResponses when needed
	balloon.Responses = nil

	// Seek to end of balloon block
	_, _ = br.Seek(balloonEndOffset, io.SeekStart)

	return balloon, nil
}

func parseResolutionDescriptor(br *gll.ByteReader) (*ResolutionDescriptor, error) {
	res := &ResolutionDescriptor{}

	// Read block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading resolution block size: %w", err)
	}

	if blockSize != 32 {
		return nil, fmt.Errorf("unexpected resolution block size: %d (expected 32)", blockSize)
	}

	// Read version check (expected: 0)
	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading resolution version check: %w", err)
	}

	if vcheck != 0 {
		return nil, fmt.Errorf("unsupported resolution version: %d", vcheck)
	}

	// Read sub-version (expected: 0)
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading resolution sub-version: %w", err)
	}

	// Read Symmetry (on-disk codes do not match enum order)
	sym, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading symmetry: %w", err)
	}

	res.Symmetry = mapSymmetryCode(sym)

	// Read FrontHalfOnly (stored as int32 boolean)
	frontHalf, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading front half only: %w", err)
	}

	res.FrontHalfOnly = frontHalf != 0

	// Read MeridianStep
	res.MeridianStep, err = br.ReadDouble()
	if err != nil {
		return nil, fmt.Errorf("reading meridian step: %w", err)
	}

	// Read ParallelStep
	res.ParallelStep, err = br.ReadDouble()
	if err != nil {
		return nil, fmt.Errorf("reading parallel step: %w", err)
	}

	return res, nil
}

// mapSymmetryCode converts on-disk symmetry codes to internal enum values.
// File mapping: 0=Axial, 1=Quarter, 2=Vertical, 3=Horizontal, 4=None.
// Internal enum order: 0=None, 1=Vertical, 2=Horizontal, 3=Quarter, 4=Axial.
func mapSymmetryCode(code int32) int32 {
	switch code {
	case 0: // Axial
		return int32(SymmetryAxial)
	case 1: // Quarter
		return int32(SymmetryQuarter)
	case 2: // Vertical
		return int32(SymmetryVertical)
	case 3: // Horizontal
		return int32(SymmetryHorizontal)
	case 4: // None
		return int32(SymmetryNone)
	default:
		return code
	}
}
