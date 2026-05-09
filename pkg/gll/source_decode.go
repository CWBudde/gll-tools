package gll

import (
	"bytes"
	"fmt"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// ParseSourceDefinitionItemBytes parses a single SourceDefinitionItem block
// from its on-disk byte representation, the inverse of the bytes produced by
// the XGLL encoder's encodeSourceDefinitionItem helper.
//
// Unlike the lazy parser used by Parse(), this helper eagerly loads
// BalloonData responses from the same byte slice so the returned item is
// fully self-contained — callers do not need to keep the source bytes around
// for a follow-up LoadBalloonResponses call.
//
// The input begins with the int32 block size, vcheck/sub-version, and item
// payload (key + SourceDefinition block).
func ParseSourceDefinitionItemBytes(data []byte) (*SourceDefinitionItem, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty source definition item bytes")
	}

	r := bytes.NewReader(data)
	br := gll.NewByteReader(r)
	item, err := parseSourceDefinitionItem(br)
	if err != nil {
		return nil, fmt.Errorf("parse source definition item: %w", err)
	}

	// Eagerly populate balloon responses so the returned item does not depend
	// on the caller retaining the original byte slice.
	if item != nil && item.Definition != nil && item.Definition.BalloonData != nil {
		balloon := item.Definition.BalloonData
		if balloon.ResponseCount > 0 && balloon.Responses == nil {
			if err := LoadBalloonResponses(r, balloon); err != nil {
				return nil, fmt.Errorf("load balloon responses: %w", err)
			}
		}
	}

	return item, nil
}
