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

// findSourceDefinitions searches for source definitions in the database area
// by looking for the pattern: count(int32) + blockSize(large) + vcheck(0) + sver(1) + keyLen(small) + key
func findSourceDefinitions(br *gll.ByteReader, startOffset, endOffset int64) ([]SourceDefinitionItem, error) {
	// Search from startOffset to endOffset for potential source definition start
	// Pattern: count (1-20), followed by large blockSize (>10000), vcheck=0, sver=0/1, small keyLen (<100)
	for offset := startOffset; offset < endOffset-100; offset += 2 {
		_, _ = br.Seek(offset, io.SeekStart)

		// Read potential count
		count, err := br.ReadInt32()
		if err != nil {
			continue
		}

		if count < 1 || count > 20 {
			continue
		}

		// Read potential item block size
		blockSize, err := br.ReadInt32()
		if err != nil {
			continue
		}
		// Source definitions are large (>10KB each typically)
		if blockSize < 10000 || blockSize > 10000000 {
			continue
		}

		// Read vcheck (should be 0)
		vcheck, err := br.ReadInt16()
		if err != nil {
			continue
		}

		if vcheck != 0 {
			continue
		}

		// Read sver (should be 0 or 1)
		sver, err := br.ReadInt16()
		if err != nil {
			continue
		}

		if sver < 0 || sver > 1 {
			continue
		}

		// Read key string length (should be small)
		keyLen, err := br.ReadInt16()
		if err != nil {
			continue
		}

		if keyLen < 1 || keyLen > 100 {
			continue
		}

		// Found potential match - seek back to count position and try parsing
		_, _ = br.Seek(offset, io.SeekStart)

		sources, err := parseSourceDefinitionBuffer(br)
		if err == nil && len(sources) > 0 {
			return sources, nil
		}
	}

	return nil, fmt.Errorf("source definitions not found")
}
