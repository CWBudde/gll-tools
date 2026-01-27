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
	return "xgllbin-pretty"
}

func (w xgllbinPrettyWriter) Write(doc *Document, out io.Writer) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	var payload bytes.Buffer
	if err := json.Indent(&payload, raw, "", "  "); err != nil {
		return fmt.Errorf("indent: %w", err)
	}

	if _, err := out.Write([]byte(xgllbinMagic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}

	if err := binary.Write(out, binary.LittleEndian, xgllbinVersion); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	if err := binary.Write(out, binary.LittleEndian, uint32(payload.Len())); err != nil {
		return fmt.Errorf("write length: %w", err)
	}

	if _, err := out.Write(payload.Bytes()); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

func init() {
	RegisterWriter(xgllbinPrettyWriter{})
}
