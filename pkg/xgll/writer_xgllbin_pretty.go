package xgll

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

type xgllbinPrettyWriter struct{}

func (w xgllbinPrettyWriter) Format() string {
	// Format key for registry
	return "xgllbin-pretty"
}

func (w xgllbinPrettyWriter) Write(doc *Document, out io.Writer) error {
	// Validate input document
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	// Encode document as JSON
	raw, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	// Pretty-print JSON payload
	var payload bytes.Buffer
	if err := json.Indent(&payload, raw, "", "  "); err != nil {
		return fmt.Errorf("indent: %w", err)
	}

	// Write magic header
	if _, err := out.Write([]byte(xgllbinMagic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}

	// Write version
	if err := binary.Write(out, binary.LittleEndian, xgllbinVersion); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	// Write payload length
	if err := binary.Write(out, binary.LittleEndian, uint32(payload.Len())); err != nil { // nolint:gosec
		return fmt.Errorf("write length: %w", err)
	}

	// Write payload bytes
	if _, err := out.Write(payload.Bytes()); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	// Done writing
	return nil
}

func init() {
	// Register pretty writer
	RegisterWriter(xgllbinPrettyWriter{})
}
