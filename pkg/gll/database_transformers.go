package gll

import (
	"fmt"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

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

// parseTransformerBuffer parses the Transformers buffer
func parseTransformerBuffer(br *gll.ByteReader, maxOffset int64) ([]Transformer, error) {
	return parseBufferItems(br, maxOffset, 0, parseTransformer)
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
	return parseBufferItems(br, maxOffset, 0, parseTapSetting)
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
