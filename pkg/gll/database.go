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

// parseBufferItems parses a standard block-wrapped buffer of items.
// The buffer layout is: blockSize(int32) + vcheck(int16) + subVersion(int16) + count(int32) + items...
// The expectedVCheck parameter specifies the expected version check value (0 for most buffers).
func parseBufferItems[T any](br *gll.ByteReader, maxOffset int64, expectedVCheck int16, parseItem func(*gll.ByteReader) (*T, error)) ([]T, error) {
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

	bufferEnd := min(br.Offset()+int64(blockSize)-4, maxOffset)

	defer func() {
		_, _ = br.Seek(bufferEnd, io.SeekStart)
	}()

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != expectedVCheck {
		return nil, fmt.Errorf("unsupported buffer version: %d (expected %d)", versionCheck, expectedVCheck)
	}

	_, err = br.ReadInt16() // sub-version
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading count: %w", err)
	}

	if count <= 0 {
		return nil, nil
	}

	items := make([]T, 0, count)

	for range count {
		if br.Offset() >= bufferEnd {
			break
		}

		item, err := parseItem(br)
		if err != nil {
			break
		}

		items = append(items, *item)
	}

	return items, nil
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
