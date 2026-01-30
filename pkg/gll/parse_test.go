package gll

import (
	"bytes"
	"encoding/binary"
	"testing"

	internalgll "github.com/cwbudde/gll-tools/internal/gll"
)

func writeString(buf *bytes.Buffer, s string) {
	// nolint:gosec
	_ = binary.Write(buf, binary.LittleEndian, int16(len(s)))
	buf.WriteString(s)
}

func buildHeaderBytes(formatVersion int16, subVersion int16) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString(internalgll.MagicEGLL)
	_ = binary.Write(buf, binary.LittleEndian, int32(0))
	writeString(buf, internalgll.MagicEASEGLL)
	_ = binary.Write(buf, binary.LittleEndian, formatVersion)

	_ = binary.Write(buf, binary.LittleEndian, subVersion)
	if formatVersion >= 4 {
		buf.Write([]byte{0x01, 0x02, 0x03, 0x04})
	}

	if formatVersion >= 6 {
		_ = binary.Write(buf, binary.LittleEndian, int32(4))
		buf.Write([]byte{0x09, 0x08, 0x07, 0x06})
	}

	return buf.Bytes()
}

func TestParseHeaderValid(t *testing.T) {
	data := buildHeaderBytes(6, 2)
	br := internalgll.NewByteReader(bytes.NewReader(data))

	file := &File{}
	if err := parseHeader(br, file); err != nil {
		t.Fatalf("parseHeader returned error: %v", err)
	}

	if file.Header.Magic != internalgll.MagicEGLL {
		t.Fatalf("magic mismatch: got %q", file.Header.Magic)
	}

	if file.Header.FormatID != internalgll.MagicEASEGLL {
		t.Fatalf("format ID mismatch: got %q", file.Header.FormatID)
	}

	if file.Header.FormatVersion != 6 || file.Header.SubVersion != 2 {
		t.Fatalf("version mismatch: got %d.%d", file.Header.FormatVersion, file.Header.SubVersion)
	}

	if file.Header.Checksum != [4]byte{0x01, 0x02, 0x03, 0x04} {
		t.Fatalf("checksum mismatch: got %v", file.Header.Checksum)
	}

	if file.Header.HashID[0] != 0x09 || file.Header.HashID[1] != 0x08 {
		t.Fatalf("hash mismatch: got %v", file.Header.HashID[:4])
	}
}

func TestParseHeaderInvalidMagic(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteString("BAD!")
	_ = binary.Write(buf, binary.LittleEndian, int32(0))
	writeString(buf, internalgll.MagicEASEGLL)
	_ = binary.Write(buf, binary.LittleEndian, int16(3))
	_ = binary.Write(buf, binary.LittleEndian, int16(0))

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))

	file := &File{}
	if err := parseHeader(br, file); err == nil {
		t.Fatalf("expected error for invalid magic")
	}
}

func TestParseHeaderInvalidFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteString(internalgll.MagicEGLL)
	_ = binary.Write(buf, binary.LittleEndian, int32(0))
	writeString(buf, "EASE_BAD")
	_ = binary.Write(buf, binary.LittleEndian, int16(3))
	_ = binary.Write(buf, binary.LittleEndian, int16(0))

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))

	file := &File{}
	if err := parseHeader(br, file); err == nil {
		t.Fatalf("expected error for invalid format ID")
	}
}

func TestParseHeaderUnsupportedVersion(t *testing.T) {
	data := buildHeaderBytes(2, 0)
	br := internalgll.NewByteReader(bytes.NewReader(data))

	file := &File{}
	if err := parseHeader(br, file); err == nil {
		t.Fatalf("expected error for unsupported version")
	}
}

func TestParseHeaderVersion3NoChecksum(t *testing.T) {
	data := buildHeaderBytes(3, 1)
	br := internalgll.NewByteReader(bytes.NewReader(data))

	file := &File{}
	if err := parseHeader(br, file); err != nil {
		t.Fatalf("parseHeader returned error: %v", err)
	}

	if file.Header.Checksum != [4]byte{} {
		t.Fatalf("checksum should be empty for v3: got %v", file.Header.Checksum)
	}

	if file.Header.HashID != [32]byte{} {
		t.Fatalf("hash should be empty for v3: got %v", file.Header.HashID)
	}
}

func TestParseHeaderVersion4Checksum(t *testing.T) {
	data := buildHeaderBytes(4, 0)
	br := internalgll.NewByteReader(bytes.NewReader(data))

	file := &File{}
	if err := parseHeader(br, file); err != nil {
		t.Fatalf("parseHeader returned error: %v", err)
	}

	if file.Header.Checksum != [4]byte{0x01, 0x02, 0x03, 0x04} {
		t.Fatalf("checksum mismatch: got %v", file.Header.Checksum)
	}

	if file.Header.HashID != [32]byte{} {
		t.Fatalf("hash should be empty for v4: got %v", file.Header.HashID)
	}
}

func TestParseHeaderHashLengthZero(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteString(internalgll.MagicEGLL)
	_ = binary.Write(buf, binary.LittleEndian, int32(0))
	writeString(buf, internalgll.MagicEASEGLL)
	_ = binary.Write(buf, binary.LittleEndian, int16(6))
	_ = binary.Write(buf, binary.LittleEndian, int16(0))
	buf.Write([]byte{0x01, 0x02, 0x03, 0x04})
	_ = binary.Write(buf, binary.LittleEndian, int32(0)) // hash length

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))

	file := &File{}
	if err := parseHeader(br, file); err != nil {
		t.Fatalf("parseHeader returned error: %v", err)
	}

	if file.Header.HashID != [32]byte{} {
		t.Fatalf("expected empty hash for zero length, got %v", file.Header.HashID)
	}
}
