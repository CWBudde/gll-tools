package xgll

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const (
	xgllbinMagic   = "XGLLBIN"
	xgllbinVersion = uint16(1)
)

type xgllbinWriter struct{}

func (w xgllbinWriter) Format() string {
	return "xgllbin"
}

func (w xgllbinWriter) Write(doc *Document, out io.Writer) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	payload, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if _, err := out.Write([]byte(xgllbinMagic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}

	if err := binary.Write(out, binary.LittleEndian, xgllbinVersion); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	if err := binary.Write(out, binary.LittleEndian, uint32(len(payload))); err != nil {
		return fmt.Errorf("write length: %w", err)
	}

	if _, err := out.Write(payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

func init() {
	RegisterWriter(xgllbinWriter{})
}
