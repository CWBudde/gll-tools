package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

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

	// RawBlock holds the original on-disk bytes of the Frame block (size
	// header + payload). Captured during Parse() so XGLL text can preserve
	// the CaseGeometry mesh + pin points verbatim via BinaryFrame.
	RawBlock []byte `json:"-"`
}

// parseFrameBuffer parses the Frames buffer
func parseFrameBuffer(br *gll.ByteReader, maxOffset int64) ([]Frame, error) {
	return parseBufferItems(br, maxOffset, 0, parseFrame)
}

func parseFrame(br *gll.ByteReader) (*Frame, error) {
	blockStart := br.Offset()

	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}

	rawBlock, _ := readRawBlock(br, blockStart, int(blockSize))

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
	if len(rawBlock) > 0 {
		frame.RawBlock = rawBlock
	}

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
	return parseBufferItems(br, maxOffset, 0, parseLabeledVector3D)
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
