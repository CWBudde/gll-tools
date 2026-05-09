package gll

import (
	"bytes"
	"fmt"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// ParseFilterGroupBytes parses a single FilterGroup block from its on-disk
// byte representation (size header + vcheck + sub-version + payload).
//
// This is the inverse of capturing the raw bytes in parseFilterGroup, and is
// used by the XGLL text decoder to inflate a `BinaryFilterGroup` base64 blob
// back into a FilterGroup struct (Label, Key, IsOverridable, Filters).
func ParseFilterGroupBytes(data []byte) (*FilterGroup, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty filter group bytes")
	}

	r := bytes.NewReader(data)
	br := gll.NewByteReader(r)
	group, err := parseFilterGroup(br)
	if err != nil {
		return nil, fmt.Errorf("parse filter group: %w", err)
	}

	return group, nil
}
