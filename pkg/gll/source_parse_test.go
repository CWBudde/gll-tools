package gll

import (
	"bytes"
	"encoding/binary"
	"testing"

	internalgll "github.com/cwbudde/gll-tools/internal/gll"
)

func TestParseResolutionDescriptor(t *testing.T) {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int32(32))
	_ = binary.Write(buf, binary.LittleEndian, int16(0))
	_ = binary.Write(buf, binary.LittleEndian, int16(0))
	_ = binary.Write(buf, binary.LittleEndian, int32(2))
	_ = binary.Write(buf, binary.LittleEndian, int32(1))
	_ = binary.Write(buf, binary.LittleEndian, float64(10))
	_ = binary.Write(buf, binary.LittleEndian, float64(5))

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))

	res, err := parseResolutionDescriptor(br)
	if err != nil {
		t.Fatalf("parseResolutionDescriptor error: %v", err)
	}

	if res.Symmetry != int32(SymmetryVertical) || !res.FrontHalfOnly || res.MeridianStep != 10 || res.ParallelStep != 5 {
		t.Fatalf("unexpected resolution descriptor: %+v", res)
	}
}

func TestResolutionDescriptorCounts(t *testing.T) {
	res := ResolutionDescriptor{MeridianStep: 10, ParallelStep: 5}
	if got := res.MeridianCount(); got != 36 {
		t.Fatalf("MeridianCount = %d, want 36", got)
	}

	if got := res.ParallelCount(); got != 37 {
		t.Fatalf("ParallelCount = %d, want 37", got)
	}

	if got := res.TotalPoints(); got != 1332 {
		t.Fatalf("TotalPoints = %d, want 1332", got)
	}
}

func TestResolutionDescriptorZeroStep(t *testing.T) {
	res := ResolutionDescriptor{}
	if res.MeridianCount() != 0 || res.ParallelCount() != 0 || res.TotalPoints() != 0 {
		t.Fatalf("expected zero counts for empty resolution descriptor")
	}
}

func TestDataTypeString(t *testing.T) {
	cases := []struct {
		value DataType
		want  string
	}{
		{DataTypeHighRes, "HighRes"},
		{DataTypeThirdOctave, "1/3 Octave"},
		{DataTypeOctave, "Octave"},
		{DataType(42), "Unknown"},
	}

	for _, tc := range cases {
		if got := tc.value.String(); got != tc.want {
			t.Fatalf("DataType.String(%d) = %q, want %q", tc.value, got, tc.want)
		}
	}
}

func TestSymmetryTypeString(t *testing.T) {
	cases := []struct {
		value SymmetryType
		want  string
	}{
		{SymmetryNone, "None"},
		{SymmetryVertical, "Vertical"},
		{SymmetryHorizontal, "Horizontal"},
		{SymmetryQuarter, "Quarter"},
		{SymmetryAxial, "Axial"},
		{SymmetryType(7), "Unknown"},
	}

	for _, tc := range cases {
		if got := tc.value.String(); got != tc.want {
			t.Fatalf("SymmetryType.String(%d) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
