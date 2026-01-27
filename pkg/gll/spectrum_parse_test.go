package gll

import (
	"bytes"
	"encoding/binary"
	"testing"

	internalgll "github.com/MeKo-Christian/gll-tools/internal/gll"
)

func TestParseRecordUncompressed(t *testing.T) {
	values := []int16{1, -2, 300}
	blockSize := int32(4 + 2 + 2 + 4 + 4 + len(values)*2)

	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, blockSize)
	_ = binary.Write(buf, binary.LittleEndian, int16(0))
	_ = binary.Write(buf, binary.LittleEndian, int16(0))
	_ = binary.Write(buf, binary.LittleEndian, int32(0))

	_ = binary.Write(buf, binary.LittleEndian, int32(len(values)))
	for _, v := range values {
		_ = binary.Write(buf, binary.LittleEndian, v)
	}

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))

	got, err := parseRecord(br)
	if err != nil {
		t.Fatalf("parseRecord error: %v", err)
	}

	if len(got) != len(values) {
		t.Fatalf("parseRecord len = %d, want %d", len(got), len(values))
	}

	for i := range values {
		if got[i] != values[i] {
			t.Fatalf("parseRecord[%d] = %d, want %d", i, got[i], values[i])
		}
	}
}

func TestParseRecordUnknownCompression(t *testing.T) {
	blockSize := int32(4 + 2 + 2 + 4 + 4)
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, blockSize)
	_ = binary.Write(buf, binary.LittleEndian, int16(0))
	_ = binary.Write(buf, binary.LittleEndian, int16(0))
	_ = binary.Write(buf, binary.LittleEndian, int32(9))
	_ = binary.Write(buf, binary.LittleEndian, int32(0))

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	if _, err := parseRecord(br); err == nil {
		t.Fatalf("expected error for unknown compression type")
	}
}

func TestParseLogSpectrumDefinition(t *testing.T) {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int32(3))
	_ = binary.Write(buf, binary.LittleEndian, float64(100))
	_ = binary.Write(buf, binary.LittleEndian, int32(4))

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	def, err := parseLogSpectrumDefinition(br)
	if err != nil {
		t.Fatalf("parseLogSpectrumDefinition error: %v", err)
	}

	if def.BandsPerOctave != 3 || def.StartFreq != 100 || def.PointCount != 4 {
		t.Fatalf("unexpected definition: %+v", def)
	}
}

func TestLogSpectrumDefinitionMethods(t *testing.T) {
	def := LogSpectrumDefinition{BandsPerOctave: 3, StartFreq: 100, PointCount: 4}
	if got := def.GetResolutionType(); got != "EThirds" {
		t.Fatalf("GetResolutionType = %q, want EThirds", got)
	}

	if got := def.GetEndFreq(); got < 199.9 || got > 200.1 {
		t.Fatalf("GetEndFreq = %f, want ~200", got)
	}

	if got := def.GetFrequency(2); got < 158.6 || got > 158.9 {
		t.Fatalf("GetFrequency(2) = %f, want ~158.7", got)
	}
}

func TestStandardBands(t *testing.T) {
	if len(Standard1_3OctaveBands) != 21 {
		t.Fatalf("expected 21 standard bands, got %d", len(Standard1_3OctaveBands))
	}

	if Standard1_3OctaveBands[0].Frequency != 50 {
		t.Fatalf("first band frequency = %f, want 50", Standard1_3OctaveBands[0].Frequency)
	}
}
