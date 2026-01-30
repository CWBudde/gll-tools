package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
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

// parseLimitBuffer parses the Limits buffer
func parseLimitBuffer(br *gll.ByteReader, maxOffset int64) ([]Limit, error) {
	return parseBufferItems(br, maxOffset, 0, parseLimit)
}

func parseLimit(br *gll.ByteReader) (*Limit, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	limit := &Limit{}

	limit.Frame, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading frame: %w", err)
	}

	typeVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading type: %w", err)
	}

	limit.Type = LimitType(typeVal)

	limit.BoxType, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading box type: %w", err)
	}

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
	return parseBufferItems(br, maxOffset, 0, parseWarning)
}

func parseWarning(br *gll.ByteReader) (*Warning, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	endOffset := br.Offset() + int64(blockSize) - 4

	versionCheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported version: %d", versionCheck)
	}

	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sub-version: %w", err)
	}

	warning := &Warning{}

	warning.Frame, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading frame: %w", err)
	}

	typeVal, err := br.ReadInt32()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading type: %w", err)
	}

	warning.Type = WarningType(typeVal)

	warning.Text, err = br.ReadString()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading text: %w", err)
	}

	warning.LimitValue, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading limit value: %w", err)
	}

	_, _ = br.Seek(endOffset, io.SeekStart)

	return warning, nil
}
