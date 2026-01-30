package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

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

// parseClusterSetupItemBuffer parses the ClusterSetups buffer
func parseClusterSetupItemBuffer(br *gll.ByteReader, maxOffset int64) ([]ClusterSetupItem, error) {
	return parseBufferItems(br, maxOffset, 0, parseClusterSetupItem)
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
	return parseBufferItems(br, maxOffset, 0, parseClusterBox)
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
