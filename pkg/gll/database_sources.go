package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// SourceFilterLink maps a source to its filter group
type SourceFilterLink struct {
	SourceKey    string `json:"source_key"`
	FilterGrpKey string `json:"filter_grp_key"`
}

// BoxInput represents an input channel on a speaker cabinet
type BoxInput struct {
	Label          string             `json:"label"`
	RatedImpedance float64            `json:"rated_impedance"` // Ohms (default 8.0)
	SourceLinks    []SourceFilterLink `json:"source_links"`    // Sources driven by this input
}

// BoxInputConfig defines input channel configurations for a box type
type BoxInputConfig struct {
	Label  string     `json:"label"`
	Key    string     `json:"key"`
	Inputs []BoxInput `json:"inputs"`
}

// parseInputConfigBuffer parses the InputConfigBuffer from a BoxType
// This buffer can be empty (block_size=0) or contain a single BoxInputConfig block
func parseInputConfigBuffer(br *gll.ByteReader, maxOffset int64) (*BoxInputConfig, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading input config buffer size: %w", err)
	}

	if blockSize <= 0 {
		// Empty buffer
		return nil, nil
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	// Parse the BoxInputConfig block
	config, err := parseBoxInputConfig(br, endOffset)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, err
	}

	_, _ = br.Seek(endOffset, io.SeekStart)
	return config, nil
}

// parseBoxInputConfig parses a BoxInputConfig block
func parseBoxInputConfig(br *gll.ByteReader, maxOffset int64) (*BoxInputConfig, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading box input config block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid box input config block size: %d", blockSize)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading box input config version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported box input config version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub_version (not used currently)
	if err != nil {
		return nil, fmt.Errorf("reading box input config sub-version: %w", err)
	}

	config := &BoxInputConfig{}

	// Read Label
	config.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading box input config label: %w", err)
	}

	// Read Key
	config.Key, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading box input config key: %w", err)
	}

	// Parse BoxInputBuffer
	inputs, err := parseBoxInputBuffer(br, endOffset)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, err
	}
	config.Inputs = inputs

	_, _ = br.Seek(endOffset, io.SeekStart)
	return config, nil
}

// parseBoxInputBuffer parses a buffer of BoxInput items
func parseBoxInputBuffer(br *gll.ByteReader, maxOffset int64) ([]BoxInput, error) {
	if br.Offset() >= maxOffset {
		return nil, nil
	}

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading box input buffer size: %w", err)
	}

	if blockSize <= 0 {
		// Empty buffer
		return nil, nil
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading box input buffer version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported box input buffer version: %d", versionCheck)
	}

	_, err = br.ReadInt16() // sub_version
	if err != nil {
		return nil, fmt.Errorf("reading box input buffer sub-version: %w", err)
	}

	// Read count
	count, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading box input count: %w", err)
	}

	if count < 0 || count > 1000 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("invalid box input count: %d", count)
	}

	inputs := make([]BoxInput, 0, count)
	for i := 0; i < int(count); i++ {
		input, err := parseBoxInput(br, endOffset)
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return nil, fmt.Errorf("parsing box input %d: %w", i, err)
		}
		inputs = append(inputs, *input)
	}

	_, _ = br.Seek(endOffset, io.SeekStart)
	return inputs, nil
}

// parseBoxInput parses a single BoxInput block
func parseBoxInput(br *gll.ByteReader, maxOffset int64) (*BoxInput, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading box input block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid box input block size: %d", blockSize)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4
	if endOffset > maxOffset {
		endOffset = maxOffset
	}

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading box input version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported box input version: %d", versionCheck)
	}

	subVersion, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading box input sub-version: %w", err)
	}

	input := &BoxInput{}

	// Read Label
	input.Label, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading box input label: %w", err)
	}

	// Read LinkCount
	linkCount, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading box input link count: %w", err)
	}

	if linkCount < 0 || linkCount > 1000 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("invalid box input link count: %d", linkCount)
	}

	// Read SourceFilterLinks (inline, no block wrappers)
	links := make([]SourceFilterLink, 0, linkCount)
	for i := 0; i < int(linkCount); i++ {
		link, err := parseSourceFilterLink(br)
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return nil, fmt.Errorf("parsing link %d: %w", i, err)
		}
		links = append(links, *link)
	}
	input.SourceLinks = links

	// Read RatedImpedance (only if sub-version >= 1)
	if subVersion >= 1 {
		input.RatedImpedance, err = br.ReadDouble()
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return nil, fmt.Errorf("reading rated impedance: %w", err)
		}
	}

	_, _ = br.Seek(endOffset, io.SeekStart)
	return input, nil
}

// parseSourceFilterLink parses a SourceFilterLink (inline, no block wrapper)
func parseSourceFilterLink(br *gll.ByteReader) (*SourceFilterLink, error) {
	link := &SourceFilterLink{}

	var err error
	link.SourceKey, err = br.ReadString()
	if err != nil {
		return nil, fmt.Errorf("reading source key: %w", err)
	}

	link.FilterGrpKey, err = br.ReadString()
	if err != nil {
		return nil, fmt.Errorf("reading filter grp key: %w", err)
	}

	return link, nil
}
