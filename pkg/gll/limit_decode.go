package gll

import (
	"bytes"
	"fmt"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// ParseLimitBytes parses a single Limit block from its on-disk byte
// representation. It is the inverse of capturing the raw bytes in
// parseLimit, used by the XGLL text decoder to inflate a BinaryLimit blob.
func ParseLimitBytes(data []byte) (*Limit, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty limit bytes")
	}

	r := bytes.NewReader(data)
	br := gll.NewByteReader(r)
	limit, err := parseLimit(br)
	if err != nil {
		return nil, fmt.Errorf("parse limit: %w", err)
	}

	return limit, nil
}

// ParseWarningBytes parses a single Warning block from its on-disk byte
// representation, mirroring ParseLimitBytes.
func ParseWarningBytes(data []byte) (*Warning, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty warning bytes")
	}

	r := bytes.NewReader(data)
	br := gll.NewByteReader(r)
	warning, err := parseWarning(br)
	if err != nil {
		return nil, fmt.Errorf("parse warning: %w", err)
	}

	return warning, nil
}

// ParseConnectorBytes parses a single Connector block from its on-disk byte
// representation. The Connector block has vcheck=1.
func ParseConnectorBytes(data []byte) (*Connector, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty connector bytes")
	}

	r := bytes.NewReader(data)
	br := gll.NewByteReader(r)
	connector, err := parseConnector(br)
	if err != nil {
		return nil, fmt.Errorf("parse connector: %w", err)
	}

	return connector, nil
}

// ParseFrameBytes parses a single Frame block from its on-disk byte
// representation. The Frame block has vcheck=1 and embeds CaseGeometry,
// pivot/center vectors, and pin points.
func ParseFrameBytes(data []byte) (*Frame, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty frame bytes")
	}

	r := bytes.NewReader(data)
	br := gll.NewByteReader(r)
	frame, err := parseFrame(br)
	if err != nil {
		return nil, fmt.Errorf("parse frame: %w", err)
	}

	return frame, nil
}
