package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// Database contains all component data from a GLL file
type Database struct {
	DataFiles         []DataFile             `json:"data_files,omitempty"`
	BoxTypes          []BoxType              `json:"box_types,omitempty"`
	Frames            []Frame                `json:"frames,omitempty"`
	Connectors        []Connector            `json:"connectors,omitempty"`
	Limits            []Limit                `json:"limits,omitempty"`
	SourceDefinitions []SourceDefinitionItem `json:"source_definitions,omitempty"`
	Warnings          []Warning              `json:"warnings,omitempty"`
	FilterGroups      []FilterGroup          `json:"filter_groups,omitempty"`
	BoxInputConfigs   []BoxInputConfig       `json:"box_input_configs,omitempty"`
	ClusterSetups     []ClusterSetupItem     `json:"cluster_setups,omitempty"`
	Presets           []GenSystemPreset      `json:"presets,omitempty"`
	IncludeFiles      []IncludeFile          `json:"include_files,omitempty"`
	AuthorFiles       []DataFile             `json:"author_files,omitempty"` // Uses same format as DataFile
	Transformers      []Transformer          `json:"transformers,omitempty"`
	SubVersion        int16                  `json:"sub_version,omitempty"` // Database sub_version for conditional parsing
	RawBlock          []byte                 `json:"raw_block,omitempty"`
}

// LabeledValueD represents a named double value (e.g., connector angles)
type LabeledValueD struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// Connector defines box-to-box connection points with splay angles
// Note: Uses vcheck=1 (not 0 like most structures)
type Connector struct {
	UpperBox string          `json:"upper_box"` // Key of upper box type
	LowerBox string          `json:"lower_box"` // Key of lower box type
	Frame    string          `json:"frame"`     // Key of rigging frame (optional)
	Angles   []LabeledValueD `json:"angles"`    // Available splay angles
}

// LabeledVector3D represents a named 3D point (e.g., pin points)
type LabeledVector3D struct {
	Label  string   `json:"label"`
	Vector Vector3D `json:"vector"`
}

// Frame represents a rigging frame for line arrays
// Note: Uses vcheck=1 (not 0 like most structures)
type Frame struct {
	Label        string            `json:"label"`
	Key          string            `json:"key"`
	TypeFlown    bool              `json:"type_flown"` // true=flown, false=ground stacked
	Weight       float64           `json:"weight"`     // Weight in kg
	CaseGeometry *CaseGeometry     `json:"case_geometry,omitempty"`
	NextPivot    *Vector3D         `json:"next_pivot,omitempty"` // Pivot point for next element
	CenterOfMass *Vector3D         `json:"center_of_mass,omitempty"`
	PinPoints    []LabeledVector3D `json:"pin_points,omitempty"` // Rigging attachment points
}

// ClusterSetupItem wraps a ClusterSetup with label and key
type ClusterSetupItem struct {
	Label string       `json:"label"`
	Key   string       `json:"key"`
	Setup ClusterSetup `json:"setup"`
}

// ClusterSetup defines a predefined array configuration
type ClusterSetup struct {
	Description string       `json:"description,omitempty"`
	Boxes       []ClusterBox `json:"boxes,omitempty"`
}

// ClusterBox represents a speaker in a cluster configuration
type ClusterBox struct {
	HashID     int32    `json:"hash_id,omitempty"`
	Label      string   `json:"label"`
	BoxTypeKey string   `json:"box_type_key"`
	Position   Vector3D `json:"position"`
	Angles     Vector3D `json:"angles"` // H, V, R rotation angles
}

// GenSystemPreset represents a system preset configuration
type GenSystemPreset struct {
	Label      string `json:"label"`
	Key        string `json:"key"`
	Config     string `json:"config,omitempty"`      // JSON-encoded config (complex structure)
	ConfigSize int    `json:"config_size,omitempty"` // Raw config byte size (unparsed)
	ConfigRaw  []byte `json:"config_raw,omitempty"`  // Raw GenSystemConfig bytes (base64 in JSON)
}

// Transformer represents a power transformer configuration
type Transformer struct {
	Label         string       `json:"label"`
	Key           string       `json:"key"`
	MaxPower      float64      `json:"max_power"`      // Max power in watts (default 4.0)
	NetVoltage    float64      `json:"net_voltage"`    // Network voltage (default 70.7V)
	LspkImpedance float64      `json:"lspk_impedance"` // Loudspeaker impedance (default 8.0 ohms)
	TapSettings   []TapSetting `json:"tap_settings,omitempty"`
}

// TapSetting represents a transformer tap setting
type TapSetting struct {
	Label      string  `json:"label"`
	Key        string  `json:"key"`
	PowerRatio float64 `json:"power_ratio"` // Power ratio (default 1.0)
}

// parseDatabase parses the database block containing all component data
//
//nolint:gocyclo // Complexity is inherent to parsing 20+ different database buffer types sequentially
func parseDatabase(br *gll.ByteReader, file *File) error {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return fmt.Errorf("invalid block size: %d", blockSize)
	}

	startOffset := br.Offset()
	blockStart := startOffset - 4
	rawBlock, _ := readRawBlock(br, blockStart, int(blockSize))
	endOffset := startOffset + int64(blockSize) - 4

	// Read version check
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return fmt.Errorf("unsupported database version: %d", versionCheck)
	}

	// Read sub-version
	subVersion, err := br.ReadInt16()
	if err != nil {
		return fmt.Errorf("reading sub-version: %w", err)
	}

	file.Database = &Database{}
	if len(rawBlock) > 0 {
		file.Database.RawBlock = rawBlock
	}

	// Database has additional header fields (8 bytes observed in sub-version 3)
	// These appear to be flags or reserved fields
	_, _ = br.ReadInt32() // Skip unknown field 1
	_, _ = br.ReadInt32() // Skip unknown field 2

	// Parse buffers in order (as per format.md)
	// 1. DataFiles (no block wrapper, just count + entries)
	dataFiles, err := parseDataFileBuffer(br, endOffset)
	if err == nil && dataFiles != nil {
		file.Database.DataFiles = dataFiles
	}

	// 2. BoxTypes (has block wrapper)
	boxTypes, err := parseBoxTypeBuffer(br, endOffset)
	if err == nil && boxTypes != nil {
		file.Database.BoxTypes = boxTypes
	}

	// 3. Frames buffer (has block wrapper, vcheck=1)
	frames, err := parseFrameBuffer(br, endOffset)
	if err == nil && len(frames) > 0 {
		file.Database.Frames = frames
	}

	// 4. Connectors buffer (has block wrapper, vcheck=1)
	connectors, err := parseConnectorBuffer(br, endOffset)
	if err == nil && len(connectors) > 0 {
		file.Database.Connectors = connectors
	}

	// 5. Limits buffer
	limits, err := parseLimitBuffer(br, endOffset)
	if err == nil && len(limits) > 0 {
		file.Database.Limits = limits
	}

	// 6. SourceDefinitions - the main acoustic data
	sources, err := parseSourceDefinitionBuffer(br)
	if err != nil || len(sources) == 0 {
		// Try fallback: search for source definitions in the file
		sources, _ = findSourceDefinitions(br, startOffset, endOffset)
	}

	if len(sources) > 0 {
		file.Database.SourceDefinitions = sources
	}

	// 7. Warnings buffer
	warnings, err := parseWarningBuffer(br, endOffset)
	if err == nil && len(warnings) > 0 {
		file.Database.Warnings = warnings
	}

	// 8. FilterGroups buffer
	filterGroups, err := parseFilterGroupBuffer(br, endOffset)
	if err == nil && len(filterGroups) > 0 {
		file.Database.FilterGroups = filterGroups
	}

	// 9. ClusterSetups buffer
	clusterSetups, err := parseClusterSetupItemBuffer(br, endOffset)
	if err == nil && len(clusterSetups) > 0 {
		file.Database.ClusterSetups = clusterSetups
	}

	// 10. Presets buffer (sub_version >= 1)
	if subVersion >= 1 {
		presets, err := parseGenSystemPresetBuffer(br, endOffset)
		if err == nil && len(presets) > 0 {
			file.Database.Presets = presets
		}
	}

	// 11-12. IncludeFiles and AuthorFiles (sub_version >= 2)
	if subVersion >= 2 {
		includeFiles, err := parseIncludeFileBuffer(br, endOffset)
		if err == nil && len(includeFiles) > 0 {
			file.Database.IncludeFiles = includeFiles
		}

		// AuthorFiles uses same format as DataFileBuffer
		authorFiles, err := parseDataFileBufferWithWrapper(br, endOffset)
		if err == nil && len(authorFiles) > 0 {
			file.Database.AuthorFiles = authorFiles
		}
	}

	// 13. Transformers buffer (sub_version >= 3)
	if subVersion >= 3 {
		transformers, err := parseTransformerBuffer(br, endOffset)
		if err == nil && len(transformers) > 0 {
			file.Database.Transformers = transformers
		}
	}

	file.Database.SubVersion = subVersion

	// Seek to end of database block
	_, err = br.Seek(endOffset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("seeking to database end: %w", err)
	}

	return nil
}

// parseConnectorBuffer parses the Connectors buffer
// Note: Connector uses vcheck=1 (not 0 like most structures)
func parseConnectorBuffer(br *gll.ByteReader, maxOffset int64) ([]Connector, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Connectors buffer has a block wrapper
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
		return nil, fmt.Errorf("reading connector count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	connectors := make([]Connector, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		connector, err := parseConnector(br)
		if err != nil {
			break
		}

		connectors = append(connectors, *connector)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return connectors, nil
}

func parseConnector(br *gll.ByteReader) (*Connector, error) {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	// Read version check - Connector uses vcheck=1
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck < 1 || versionCheck > 1 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported connector version: %d", versionCheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	connector := &Connector{}

	// Read UpperBox
	connector.UpperBox, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading upper box: %w", err)
	}

	// Read LowerBox
	connector.LowerBox, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading lower box: %w", err)
	}

	// Read Frame
	connector.Frame, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading frame: %w", err)
	}

	// Read Angles (LabeledValueDBuffer)
	angles, err := parseLabeledValueDBuffer(br, endOffset)
	if err == nil {
		connector.Angles = angles
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return connector, nil
}

// parseLabeledValueDBuffer parses a buffer of labeled double values
func parseLabeledValueDBuffer(br *gll.ByteReader, maxOffset int64) ([]LabeledValueD, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
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
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	values := make([]LabeledValueD, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		lv, err := parseLabeledValueD(br)
		if err != nil {
			break
		}

		values = append(values, *lv)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return values, nil
}

func parseLabeledValueD(br *gll.ByteReader) (*LabeledValueD, error) {
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

	lv := &LabeledValueD{}

	// Read Label
	lv.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read Value
	lv.Value, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading value: %w", err)
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return lv, nil
}

// parseFrameBuffer parses the Frames buffer
// Note: Frame uses vcheck=1 (not 0 like most structures)
func parseFrameBuffer(br *gll.ByteReader, maxOffset int64) ([]Frame, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Frames buffer has a block wrapper
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
		return nil, fmt.Errorf("reading frame count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	frames := make([]Frame, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		frame, err := parseFrame(br)
		if err != nil {
			break
		}

		frames = append(frames, *frame)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return frames, nil
}

func parseFrame(br *gll.ByteReader) (*Frame, error) {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	// Read version check - Frame uses vcheck=1
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck < 1 || versionCheck > 1 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported frame version: %d", versionCheck)
	}

	// Read sub-version
	subVersion, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	frame := &Frame{}

	// Read Label
	frame.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read TypeFlown (int32 as bool)
	typeFlownVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading type flown: %w", err)
	}

	frame.TypeFlown = typeFlownVal != 0

	// Parse CaseGeometry (3D mesh data)
	caseGeometry, err := parseCaseGeometry(br, endOffset)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return frame, nil // Return what we have
	}
	frame.CaseGeometry = caseGeometry

	// For sub_version 0, there are extra padding fields
	if subVersion == 0 {
		_, _ = br.ReadInt32() // Skip padding
		_, _ = br.ReadInt32() // Skip padding
	}

	// Read NextPivot (Vector3D)
	nextPivot, err := parseVector3D(br)
	if err == nil {
		frame.NextPivot = nextPivot
	}

	// For sub_version 0, there are extra padding fields
	if subVersion == 0 {
		_, _ = br.ReadInt32() // Skip padding
		_, _ = br.ReadInt32() // Skip padding
	}

	// Read CenterOfMass (Vector3D)
	centerOfMass, err := parseVector3D(br)
	if err == nil {
		frame.CenterOfMass = centerOfMass
	}

	// Read Weight
	frame.Weight, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return frame, nil // Return what we have
	}

	// Read PinPoints (LabeledVector3DBuffer)
	pinPoints, err := parseLabeledVector3DBuffer(br, endOffset)
	if err == nil {
		frame.PinPoints = pinPoints
	}

	// Read Key (at the end in Frame)
	frame.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return frame, nil // Return what we have
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return frame, nil
}

// parseLabeledVector3DBuffer parses a buffer of labeled 3D vectors
func parseLabeledVector3DBuffer(br *gll.ByteReader, maxOffset int64) ([]LabeledVector3D, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
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
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	vectors := make([]LabeledVector3D, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		lv, err := parseLabeledVector3D(br)
		if err != nil {
			break
		}

		vectors = append(vectors, *lv)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return vectors, nil
}

func parseLabeledVector3D(br *gll.ByteReader) (*LabeledVector3D, error) {
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

	lv := &LabeledVector3D{}

	// Read Label
	lv.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// For sub_version 0, skip two int32 padding fields
	if subVersion == 0 {
		_, _ = br.ReadInt32()
		_, _ = br.ReadInt32()
	}

	// Read Vector3D (3 doubles, no block wrapper)
	vec, err := parseVector3D(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading vector: %w", err)
	}

	lv.Vector = *vec

	_, _ = br.Seek(endOffset, io.SeekStart)

	return lv, nil
}

// parseVector3D reads a 3D vector from the stream
func parseVector3D(br *gll.ByteReader) (*Vector3D, error) {
	v := &Vector3D{}

	var err error

	v.X, err = br.ReadDouble()
	if err != nil {
		return nil, err
	}

	v.Y, err = br.ReadDouble()
	if err != nil {
		return nil, err
	}

	v.Z, err = br.ReadDouble()
	if err != nil {
		return nil, err
	}

	return v, nil
}

// parseClusterSetupItemBuffer parses the ClusterSetups buffer
func parseClusterSetupItemBuffer(br *gll.ByteReader, maxOffset int64) ([]ClusterSetupItem, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
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
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	items := make([]ClusterSetupItem, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		item, err := parseClusterSetupItem(br)
		if err != nil {
			break
		}

		items = append(items, *item)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return items, nil
}

func parseClusterSetupItem(br *gll.ByteReader) (*ClusterSetupItem, error) {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	// Read version check - ClusterSetupItem uses vcheck 0 or 1
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck < 0 || versionCheck > 1 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	item := &ClusterSetupItem{}

	// Read Label
	item.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read Key
	item.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading key: %w", err)
	}

	// Parse ClusterSetup (different format based on versionCheck)
	if versionCheck == 0 {
		// Old format: ClusterBoxBuffer directly
		boxes, err := parseClusterBoxBuffer(br, endOffset)
		if err == nil {
			item.Setup.Boxes = boxes
		}
	} else {
		// New format: full ClusterSetup
		setup, err := parseClusterSetup(br, endOffset)
		if err == nil {
			item.Setup = *setup
		}
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return item, nil
}

func parseClusterSetup(br *gll.ByteReader, maxOffset int64) (*ClusterSetup, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

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

	setup := &ClusterSetup{}

	// Read Description
	setup.Description, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading description: %w", err)
	}

	// Parse ClusterBoxBuffer
	boxes, err := parseClusterBoxBuffer(br, endOffset)
	if err == nil {
		setup.Boxes = boxes
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return setup, nil
}

func parseClusterBoxBuffer(br *gll.ByteReader, maxOffset int64) ([]ClusterBox, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
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
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	boxes := make([]ClusterBox, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		box, err := parseClusterBox(br)
		if err != nil {
			break
		}

		boxes = append(boxes, *box)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return boxes, nil
}

func parseClusterBox(br *gll.ByteReader) (*ClusterBox, error) {
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

	box := &ClusterBox{}

	// Read int4 and int5 (unknown fields)
	_, _ = br.ReadInt32() // int4
	_, _ = br.ReadInt32() // int5

	// Read HashID
	hashID, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading hash ID: %w", err)
	}
	box.HashID = hashID

	// Read Label
	box.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Skip double (num)
	_, _ = br.ReadDouble()

	// Read BoxTypeKey
	box.BoxTypeKey, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading box type key: %w", err)
	}

	// For sub_version 0, skip padding
	if subVersion == 0 {
		_, _ = br.ReadInt32()
		_, _ = br.ReadInt32()
		_, _ = br.ReadInt32()
		_, _ = br.ReadInt32()
	}

	// Read Position (Vector3D as 3 doubles)
	box.Position.X, _ = br.ReadDouble()
	box.Position.Y, _ = br.ReadDouble()
	box.Position.Z, _ = br.ReadDouble()

	// For sub_version 0, skip padding
	if subVersion == 0 {
		_, _ = br.ReadInt32()
		_, _ = br.ReadInt32()
	}

	// Read Angles (Vector3D as 3 doubles)
	box.Angles.X, _ = br.ReadDouble()
	box.Angles.Y, _ = br.ReadDouble()
	box.Angles.Z, _ = br.ReadDouble()

	_, _ = br.Seek(endOffset, io.SeekStart)

	return box, nil
}

// parseGenSystemPresetBuffer parses the Presets buffer
func parseGenSystemPresetBuffer(br *gll.ByteReader, maxOffset int64) ([]GenSystemPreset, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
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
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	presets := make([]GenSystemPreset, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		preset, err := parseGenSystemPreset(br)
		if err != nil {
			break
		}

		presets = append(presets, *preset)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return presets, nil
}

func parseGenSystemPreset(br *gll.ByteReader) (*GenSystemPreset, error) {
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

	preset := &GenSystemPreset{}

	// Read Label
	preset.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read Key
	preset.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading key: %w", err)
	}

	// Skip GenSystemConfig - it's complex; capture raw size/bytes for diagnostics
	if remaining := int(endOffset - br.Offset()); remaining > 0 {
		preset.ConfigSize = remaining
		raw, err := br.ReadBytes(remaining)
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return nil, fmt.Errorf("reading config raw bytes: %w", err)
		}
		preset.ConfigRaw = raw
		// ReadBytes already advances offset; avoid seeking again here.
		_, _ = br.Seek(endOffset, io.SeekStart)
		return preset, nil
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return preset, nil
}

// parseTransformerBuffer parses the Transformers buffer
func parseTransformerBuffer(br *gll.ByteReader, maxOffset int64) ([]Transformer, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
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
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	transformers := make([]Transformer, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		t, err := parseTransformer(br)
		if err != nil {
			break
		}

		transformers = append(transformers, *t)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return transformers, nil
}

func parseTransformer(br *gll.ByteReader) (*Transformer, error) {
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

	t := &Transformer{
		MaxPower:      4.0,  // Default
		NetVoltage:    70.7, // Default
		LspkImpedance: 8.0,  // Default
	}

	// Read Label
	t.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read Key
	t.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading key: %w", err)
	}

	// Read MaxPower
	t.MaxPower, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading max power: %w", err)
	}

	// Read NetVoltage
	t.NetVoltage, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading net voltage: %w", err)
	}

	// Read LspkImpedance
	t.LspkImpedance, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading lspk impedance: %w", err)
	}

	// Parse TapSettingBuffer
	taps, err := parseTapSettingBuffer(br, endOffset)
	if err == nil {
		t.TapSettings = taps
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return t, nil
}

func parseTapSettingBuffer(br *gll.ByteReader, maxOffset int64) ([]TapSetting, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	// Buffer has block wrapper
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
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
		return nil, nil
	}

	taps := make([]TapSetting, 0, count)

	for i := int32(0); i < count; i++ {
		if br.Offset() >= bufferEnd {
			break
		}

		tap, err := parseTapSetting(br)
		if err != nil {
			break
		}

		taps = append(taps, *tap)
	}

	_, _ = br.Seek(bufferEnd, io.SeekStart)

	return taps, nil
}

func parseTapSetting(br *gll.ByteReader) (*TapSetting, error) {
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

	tap := &TapSetting{
		PowerRatio: 1.0, // Default
	}

	// Read Label
	tap.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading label: %w", err)
	}

	// Read Key
	tap.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading key: %w", err)
	}

	// Read PowerRatio
	tap.PowerRatio, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading power ratio: %w", err)
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return tap, nil
}
