package gll

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// GenSystemConfig represents a decoded preset configuration (partial).
type GenSystemConfig struct {
	VersionCheck            int16                    `json:"vcheck,omitempty"`
	SubVersion              int16                    `json:"sver,omitempty"`
	GridAngle               float64                  `json:"grid_angle"`
	UnknownInt32            int32                    `json:"unknown_int32"`
	FrameKey                string                   `json:"frame_key"`
	ClusterSetupKey         string                   `json:"cluster_setup_key"`
	Elements                []GenSystemConfigElement `json:"elements,omitempty"`
	SystemType              int32                    `json:"system_type"`
	UserClusterSetupPresent bool                     `json:"user_cluster_setup_present,omitempty"`
	UserClusterSetupSize    int                      `json:"user_cluster_setup_size,omitempty"`
}

// GenSystemConfigElement represents one configured element in a preset.
type GenSystemConfigElement struct {
	BoxTypeKey      string               `json:"box_type_key"`
	SplayAngle      float64              `json:"splay_angle"`
	Gain            float64              `json:"gain"`
	InputConfigKey  string               `json:"input_config_key"`
	Sources         int32                `json:"sources"`
	FilterDefKeys   []string             `json:"filter_def_keys,omitempty"`
	InternalFilters []*GenericFilterBank `json:"internal_filters,omitempty"`
	ExternalFilters []*GenericFilterBank `json:"external_filters,omitempty"`
	OverrideFlags   []bool               `json:"override_flags,omitempty"`
	OverrideLabels  []string             `json:"override_labels,omitempty"`
	OverrideFilters []*GenericFilterBank `json:"override_filters,omitempty"`
}

// DecodeGenSystemConfigRaw decodes a GenSystemConfig from raw preset bytes.
// This parser focuses on top-level fields and element metadata and skips
// embedded filter bank contents.
func DecodeGenSystemConfigRaw(data []byte) (*GenSystemConfig, error) {
	br := gll.NewByteReader(bytes.NewReader(data))

	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading block size: %w", err)
	}
	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size: %d", blockSize)
	}
	endOffset := br.Offset() + int64(blockSize) - 4

	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading vcheck: %w", err)
	}
	if vcheck != 0 {
		return nil, fmt.Errorf("unsupported vcheck: %d", vcheck)
	}

	sver, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading sver: %w", err)
	}

	cfg := &GenSystemConfig{
		VersionCheck: vcheck,
		SubVersion:   sver,
	}

	if sver >= 0 {
		cfg.GridAngle, err = br.ReadDouble()
		if err != nil {
			return nil, fmt.Errorf("reading grid angle: %w", err)
		}

		cfg.UnknownInt32, err = br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("reading unknown int32: %w", err)
		}

		cfg.FrameKey, err = br.ReadString()
		if err != nil {
			return nil, fmt.Errorf("reading frame key: %w", err)
		}

		cfg.ClusterSetupKey, err = br.ReadString()
		if err != nil {
			return nil, fmt.Errorf("reading cluster setup key: %w", err)
		}

		cfg.Elements, err = parseGenSystemConfigElementBuffer(br)
		if err != nil {
			return nil, err
		}

		if sver >= 1 {
			cfg.SystemType, err = br.ReadInt32()
			if err != nil {
				return nil, fmt.Errorf("reading system type: %w", err)
			}

			if cfg.SystemType == 1 && cfg.ClusterSetupKey == "" {
				// User-defined cluster setup follows.
				cfg.UserClusterSetupPresent = true
				size, err := skipBlockRaw(br)
				if err != nil {
					return nil, fmt.Errorf("skipping user cluster setup: %w", err)
				}
				cfg.UserClusterSetupSize = size
			}
		}
	}

	_, _ = br.Seek(endOffset, io.SeekStart)
	return cfg, nil
}

func parseGenSystemConfigElementBuffer(br *gll.ByteReader) ([]GenSystemConfigElement, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading element buffer size: %w", err)
	}
	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid element buffer size: %d", blockSize)
	}
	endOffset := br.Offset() + int64(blockSize) - 4

	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading element buffer vcheck: %w", err)
	}
	if vcheck != 0 {
		return nil, fmt.Errorf("unsupported element buffer vcheck: %d", vcheck)
	}

	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading element buffer sver: %w", err)
	}

	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading element count: %w", err)
	}
	if count < 0 {
		return nil, fmt.Errorf("invalid element count: %d", count)
	}

	elements := make([]GenSystemConfigElement, 0, count)
	for i := 0; i < int(count); i++ {
		elem, err := parseGenSystemConfigElement(br)
		if err != nil {
			return nil, fmt.Errorf("parsing element %d: %w", i, err)
		}
		elements = append(elements, *elem)
	}

	_, _ = br.Seek(endOffset, io.SeekStart)
	return elements, nil
}

func parseGenSystemConfigElement(br *gll.ByteReader) (*GenSystemConfigElement, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading element block size: %w", err)
	}
	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid element block size: %d", blockSize)
	}
	endOffset := br.Offset() + int64(blockSize) - 4

	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading element vcheck: %w", err)
	}
	if vcheck != 0 {
		return nil, fmt.Errorf("unsupported element vcheck: %d", vcheck)
	}

	sver, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading element sver: %w", err)
	}

	elem := &GenSystemConfigElement{}
	if sver >= 0 {
		elem.BoxTypeKey, err = br.ReadString()
		if err != nil {
			return nil, fmt.Errorf("reading box type key: %w", err)
		}

		elem.SplayAngle, err = br.ReadDouble()
		if err != nil {
			return nil, fmt.Errorf("reading splay angle: %w", err)
		}

		elem.Gain, err = br.ReadDouble()
		if err != nil {
			return nil, fmt.Errorf("reading gain: %w", err)
		}

		elem.InputConfigKey, err = br.ReadString()
		if err != nil {
			return nil, fmt.Errorf("reading input config key: %w", err)
		}

		elem.Sources, err = br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("reading source count: %w", err)
		}
		if elem.Sources < 0 {
			return nil, fmt.Errorf("invalid source count: %d", elem.Sources)
		}

		for i := 0; i < int(elem.Sources); i++ {
			filterKey, err := br.ReadString()
			if err != nil {
				return nil, fmt.Errorf("reading filter def key: %w", err)
			}
			elem.FilterDefKeys = append(elem.FilterDefKeys, filterKey)

			internalBank, err := parseGenericFilterBank(br, endOffset)
			if err != nil {
				return nil, fmt.Errorf("parsing internal filter bank: %w", err)
			}
			elem.InternalFilters = append(elem.InternalFilters, internalBank)

			externalBank, err := parseGenericFilterBank(br, endOffset)
			if err != nil {
				return nil, fmt.Errorf("parsing external filter bank: %w", err)
			}
			elem.ExternalFilters = append(elem.ExternalFilters, externalBank)
		}

		if sver >= 1 {
			for i := 0; i < int(elem.Sources); i++ {
				flag, err := br.ReadInt16()
				if err != nil {
					return nil, fmt.Errorf("reading override flag: %w", err)
				}
				elem.OverrideFlags = append(elem.OverrideFlags, (flag&1) != 0)

				label, err := br.ReadString()
				if err != nil {
					return nil, fmt.Errorf("reading override label: %w", err)
				}
				elem.OverrideLabels = append(elem.OverrideLabels, label)

				overrideBank, err := parseGenericFilterBank(br, endOffset)
				if err != nil {
					return nil, fmt.Errorf("parsing override filter bank: %w", err)
				}
				elem.OverrideFilters = append(elem.OverrideFilters, overrideBank)
			}
		}
	}

	_, _ = br.Seek(endOffset, io.SeekStart)
	return elem, nil
}

func skipBlockRaw(br *gll.ByteReader) (int, error) {
	blockSize, err := br.ReadInt32()
	if err != nil {
		return 0, err
	}
	if blockSize <= 0 {
		return 0, fmt.Errorf("invalid block size: %d", blockSize)
	}
	endOffset := br.Offset() + int64(blockSize) - 4
	_, err = br.Seek(endOffset, io.SeekStart)
	return int(blockSize), err
}

// SplayAngleDeg returns the splay angle in degrees for display.
func (e GenSystemConfigElement) SplayAngleDeg() float64 {
	return e.SplayAngle * (180.0 / math.Pi)
}
