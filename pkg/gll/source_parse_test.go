package gll

import (
	"bytes"
	"encoding/binary"
	"testing"

	internalgll "github.com/cwbudde/gll-tools/internal/gll"
)

func TestParseResolutionDescriptor(t *testing.T) {
	// Build a minimal ResolutionDescriptor block
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int32(32))
	_ = binary.Write(buf, binary.LittleEndian, int16(0))
	_ = binary.Write(buf, binary.LittleEndian, int16(0))
	_ = binary.Write(buf, binary.LittleEndian, int32(2))
	_ = binary.Write(buf, binary.LittleEndian, int32(1))
	_ = binary.Write(buf, binary.LittleEndian, float64(10))
	_ = binary.Write(buf, binary.LittleEndian, float64(5))

	// Parse from byte reader
	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))

	res, err := parseResolutionDescriptor(br)
	if err != nil {
		t.Fatalf("parseResolutionDescriptor error: %v", err)
	}

	// Verify parsed descriptor
	if res.Symmetry != int32(SymmetryVertical) || !res.FrontHalfOnly || res.MeridianStep != 10 || res.ParallelStep != 5 {
		t.Fatalf("unexpected resolution descriptor: %+v", res)
	}
}

func TestResolutionDescriptorCounts(t *testing.T) {
	// Check derived counts for known steps
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
	// Zero steps should yield zero counts
	res := ResolutionDescriptor{}
	if res.MeridianCount() != 0 || res.ParallelCount() != 0 || res.TotalPoints() != 0 {
		t.Fatalf("expected zero counts for empty resolution descriptor")
	}
}

func TestDataTypeString(t *testing.T) {
	// Validate DataType string formatting
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
	// Validate SymmetryType string formatting
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

func TestSkipBlock(t *testing.T) {
	t.Run("skips block and leaves trailing bytes intact", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(12)) // size includes 4
		buf.Write(bytes.Repeat([]byte{0xAA}, 8))              // 8 bytes payload
		buf.WriteByte(0x42)                                   // trailing sentinel

		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if err := skipBlock(br, int64(buf.Len())); err != nil {
			t.Fatalf("skipBlock: %v", err)
		}
		next, err := br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte after skip: %v", err)
		}
		if next != 0x42 {
			t.Errorf("byte after skip = 0x%X, want 0x42", next)
		}
	})

	t.Run("zero block size returns nil without seeking", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if err := skipBlock(br, 100); err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})

	t.Run("propagates short read", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader(nil))
		if err := skipBlock(br, 100); err == nil {
			t.Error("want error on empty reader, got nil")
		}
	})

	t.Run("clamps endOffset to maxOffset", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(100)) // claims more than available
		buf.Write(bytes.Repeat([]byte{0}, 6))
		// Use io.SeekEnd-style limit: maxOffset smaller than claimed end.
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if err := skipBlock(br, 8); err != nil {
			t.Errorf("err = %v, want nil (clamp)", err)
		}
	})
}

func TestParseSourceFilterLink(t *testing.T) {
	build := func(srcKey, filterKey string) []byte {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int16(len(srcKey))) //nolint:gosec // bounded by literal len
		buf.WriteString(srcKey)
		_ = binary.Write(buf, binary.LittleEndian, int16(len(filterKey))) //nolint:gosec // bounded by literal len
		buf.WriteString(filterKey)
		return buf.Bytes()
	}

	t.Run("happy path", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader(build("srcA", "fltX")))
		link, err := parseSourceFilterLink(br)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if link.SourceKey != "srcA" || link.FilterGrpKey != "fltX" {
			t.Errorf("got %+v, want {srcA, fltX}", link)
		}
	})

	t.Run("missing filter key", func(t *testing.T) {
		// SourceKey then truncated.
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int16(2))
		buf.WriteString("ok")
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		_, err := parseSourceFilterLink(br)
		if err == nil {
			t.Fatal("want error on truncated input")
		}
	})

	t.Run("empty input returns error", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader(nil))
		_, err := parseSourceFilterLink(br)
		if err == nil {
			t.Fatal("want error on empty input")
		}
	})
}
