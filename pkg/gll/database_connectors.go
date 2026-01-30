package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

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

// parseConnectorBuffer parses the Connectors buffer
func parseConnectorBuffer(br *gll.ByteReader, maxOffset int64) ([]Connector, error) {
	return parseBufferItems(br, maxOffset, 0, parseConnector)
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
	return parseBufferItems(br, maxOffset, 0, parseLabeledValueD)
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
