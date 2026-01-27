package gll

import (
	"bytes"
	"encoding/binary"
	"testing"

	internalgll "github.com/MeKo-Christian/gll-tools/internal/gll"
)

func writeTestString(buf *bytes.Buffer, s string) {
	_ = binary.Write(buf, binary.LittleEndian, int16(len(s)))
	_, _ = buf.WriteString(s)
}

func buildMinimalDatabaseBlock() []byte {
	body := &bytes.Buffer{}
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // version check
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // sub-version
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // unknown1
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // unknown2
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // data files count
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // box types block size
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // frames block size
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // connectors block size
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // limits block size
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // source definitions count
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // warnings block size
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // filter groups block size
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // cluster setups block size

	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int32(4+body.Len()))
	_, _ = buf.Write(body.Bytes())

	return buf.Bytes()
}

func buildGenSystemBlock() []byte {
	body := &bytes.Buffer{}
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // version check
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // sub-version

	writeTestString(body, "Label")
	_ = binary.Write(body, binary.LittleEndian, float64(1.5))
	writeTestString(body, "Key")
	_ = binary.Write(body, binary.LittleEndian, int32(SystemTypeLineArray))
	writeTestString(body, "Company")
	writeTestString(body, "Info")
	writeTestString(body, "Copyright")
	writeTestString(body, "Support")
	writeTestString(body, "Website")
	writeTestString(body, "Email")
	_ = binary.Write(body, binary.LittleEndian, int32(42))

	_, _ = body.Write(buildMinimalDatabaseBlock())

	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int32(4+body.Len()))
	_, _ = buf.Write(body.Bytes())

	return buf.Bytes()
}

func TestParseGenSystemFields(t *testing.T) {
	br := internalgll.NewByteReader(bytes.NewReader(buildGenSystemBlock()))
	file := &File{}

	if err := parseGenSystem(br, file); err != nil {
		t.Fatalf("parseGenSystem error: %v", err)
	}

	if file.GenSystem.Label != "Label" {
		t.Fatalf("Label = %q, want %q", file.GenSystem.Label, "Label")
	}

	if file.GenSystem.Key != "Key" {
		t.Fatalf("Key = %q, want %q", file.GenSystem.Key, "Key")
	}

	if file.GenSystem.Company != "Company" || file.GenSystem.InfoText != "Info" {
		t.Fatalf("metadata mismatch: %+v", file.GenSystem)
	}

	if file.GenSystem.Type != SystemTypeLineArray {
		t.Fatalf("Type = %v, want %v", file.GenSystem.Type, SystemTypeLineArray)
	}

	if file.GenSystem.BackgroundColor != 42 {
		t.Fatalf("BackgroundColor = %d, want 42", file.GenSystem.BackgroundColor)
	}
}
