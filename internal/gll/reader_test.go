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

func TestReadInt32(t *testing.T) {
	cases := []struct {
		v    int32
		want int32
	}{
		{0, 0},
		{1, 1},
		{-1, -1},
		{1000000, 1000000},
		{-2147483648, -2147483648},
	}
	for _, tc := range cases {
		buf := &bytes.Buffer{}
		_ = binary.Write(buf, binary.LittleEndian, tc.v)
		br := NewByteReader(bytes.NewReader(buf.Bytes()))
		got, err := br.ReadInt32()
		if err != nil {
			t.Fatalf("ReadInt32(%d) error: %v", tc.v, err)
		}
		if got != tc.want {
			t.Errorf("ReadInt32(%d) = %d, want %d", tc.v, got, tc.want)
		}
	}
}

func TestReadInt32EOF(t *testing.T) {
	br := NewByteReader(bytes.NewReader([]byte{0x01})) // only 1 byte, need 4
	_, err := br.ReadInt32()
	if err == nil {
		t.Fatal("expected error reading truncated int32")
	}
}

func TestReadDouble(t *testing.T) {
	cases := []float64{0.0, 1.0, -1.0, 3.141592653589793, 1e100, -1e-100}
	for _, v := range cases {
		buf := &bytes.Buffer{}
		_ = binary.Write(buf, binary.LittleEndian, v)
		br := NewByteReader(bytes.NewReader(buf.Bytes()))
		got, err := br.ReadDouble()
		if err != nil {
			t.Fatalf("ReadDouble(%v) error: %v", v, err)
		}
		if got != v {
			t.Errorf("ReadDouble(%v) = %v, want %v", v, got, v)
		}
	}
}

func TestReadDoubleEOF(t *testing.T) {
	br := NewByteReader(bytes.NewReader([]byte{0x01, 0x02, 0x03})) // only 3 bytes, need 8
	_, err := br.ReadDouble()
	if err == nil {
		t.Fatal("expected error reading truncated float64")
	}
}

func TestReadSingle(t *testing.T) {
	cases := []float32{0.0, 1.0, -1.0, 3.14}
	for _, v := range cases {
		buf := &bytes.Buffer{}
		_ = binary.Write(buf, binary.LittleEndian, v)
		br := NewByteReader(bytes.NewReader(buf.Bytes()))
		got, err := br.ReadSingle()
		if err != nil {
			t.Fatalf("ReadSingle(%v) error: %v", v, err)
		}
		if got != v {
			t.Errorf("ReadSingle(%v) = %v, want %v", v, got, v)
		}
	}
}

func TestReadByte(t *testing.T) {
	data := []byte{0x00, 0xFF, 0x42}
	br := NewByteReader(bytes.NewReader(data))
	for i, want := range data {
		got, err := br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte[%d] error: %v", i, err)
		}
		if got != want {
			t.Errorf("ReadByte[%d] = 0x%02x, want 0x%02x", i, got, want)
		}
	}
	// Next read should fail (EOF)
	_, err := br.ReadByte()
	if err == nil {
		t.Fatal("expected EOF error after last byte")
	}
}

func TestOffset(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	br := NewByteReader(bytes.NewReader(data))

	if br.Offset() != 0 {
		t.Errorf("initial Offset = %d, want 0", br.Offset())
	}

	_, _ = br.ReadByte()
	if br.Offset() != 1 {
		t.Errorf("after ReadByte, Offset = %d, want 1", br.Offset())
	}

	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int32(42))
	br2 := NewByteReader(bytes.NewReader(buf.Bytes()))
	_, _ = br2.ReadInt32()
	if br2.Offset() != 4 {
		t.Errorf("after ReadInt32, Offset = %d, want 4", br2.Offset())
	}
}

func TestSeek(t *testing.T) {
	data := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE}
	br := NewByteReader(bytes.NewReader(data))

	_, err := br.Seek(2, 0) // io.SeekStart
	if err != nil {
		t.Fatalf("Seek error: %v", err)
	}
	if br.Offset() != 2 {
		t.Errorf("after Seek(2), Offset = %d, want 2", br.Offset())
	}

	b, err := br.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte after Seek error: %v", err)
	}
	if b != 0xCC {
		t.Errorf("byte after Seek(2) = 0x%02x, want 0xCC", b)
	}
	if br.Offset() != 3 {
		t.Errorf("after ReadByte at 2, Offset = %d, want 3", br.Offset())
	}
}

func TestMin(t *testing.T) {
	cases := []struct{ a, b, want int }{
		{0, 0, 0},
		{1, 2, 1},
		{2, 1, 1},
		{-5, 3, -5},
		{100, 100, 100},
	}
	for _, tc := range cases {
		got := Min(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("Min(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
