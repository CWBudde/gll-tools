package xgll

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const (
	// Binary container header
	xgllbinMagic   = "XGLLBIN"
	xgllbinVersion = uint16(1)
)

type xgllbinWriter struct{}

func (w xgllbinWriter) Format() string {
	// Format key for registry
	return "xgllbin"
}

func (w xgllbinWriter) Write(doc *Document, out io.Writer) error {
	// Validate input document
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	// Serialize document as JSON payload
	payload, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	// Write magic header
	if _, err := out.Write([]byte(xgllbinMagic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}

	// Write binary version
	if err := binary.Write(out, binary.LittleEndian, xgllbinVersion); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	// Write payload length
	if err := binary.Write(out, binary.LittleEndian, uint32(len(payload))); err != nil { // nolint:gosec
		return fmt.Errorf("write length: %w", err)
	}

	// Write payload bytes
	if _, err := out.Write(payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	// Done writing
	return nil
}

func init() {
	// Register XGLL binary writer
	RegisterWriter(xgllbinWriter{})
}
