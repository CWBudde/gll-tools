package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// GenSystemPreset represents a system preset configuration
type GenSystemPreset struct {
	Label      string `json:"label"`
	Key        string `json:"key"`
	Config     string `json:"config,omitempty"`      // JSON-encoded config (complex structure)
	ConfigSize int    `json:"config_size,omitempty"` // Raw config byte size (unparsed)
	ConfigRaw  []byte `json:"config_raw,omitempty"`  // Raw GenSystemConfig bytes (base64 in JSON)
}

// parseGenSystemPresetBuffer parses the Presets buffer
func parseGenSystemPresetBuffer(br *gll.ByteReader, maxOffset int64) ([]GenSystemPreset, error) {
	return parseBufferItems(br, maxOffset, 0, parseGenSystemPreset)
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
