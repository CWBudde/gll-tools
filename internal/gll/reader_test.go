package gll

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestReadStringEmpty(t *testing.T) {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int16(0))

	br := NewByteReader(bytes.NewReader(buf.Bytes()))
	got, err := br.ReadString()
	if err != nil {
		t.Fatalf("ReadString error: %v", err)
	}

	if got != "" {
		t.Fatalf("ReadString = %q, want empty", got)
	}
}

func TestReadStringNegativeLength(t *testing.T) {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int16(-1))

	br := NewByteReader(bytes.NewReader(buf.Bytes()))
	_, err := br.ReadString()
	if err == nil {
		t.Fatal("expected error for negative length")
	}
}

func TestReadStringValue(t *testing.T) {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int16(4))
	buf.WriteString("test")

	br := NewByteReader(bytes.NewReader(buf.Bytes()))
	got, err := br.ReadString()
	if err != nil {
		t.Fatalf("ReadString error: %v", err)
	}

	if got != "test" {
		t.Fatalf("ReadString = %q, want %q", got, "test")
	}
}
