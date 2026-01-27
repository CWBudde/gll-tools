package gll

import (
	"testing"
)

func TestReadBitsFromBytes(t *testing.T) {
	// Test reading bits from a byte array
	// 0b10110100 0b00001111 = 0xB4 0x0F
	data := []byte{0xB4, 0x0F}

	cases := []struct {
		bitPos int
		nBits  int
		want   int16
	}{
		{0, 1, 0},      // bit 0 of 0xB4 (10110100) = 0
		{1, 1, 0},      // bit 1 of 0xB4 = 0
		{2, 1, 1},      // bit 2 of 0xB4 = 1
		{0, 4, 4},      // bits 0-3: 0100 = 4
		{4, 4, 11},     // bits 4-7: 1011 = 11
		{0, 8, 0xB4},   // all 8 bits of first byte
		{8, 4, 0x0F},   // bits 8-11: 1111 = 15
		{4, 8, 0xFB},   // bits 4-11 crossing byte boundary
		{0, 16, 0xFB4}, // all 16 bits
	}

	for _, tc := range cases {
		got := readBitsFromBytes(data, tc.bitPos, tc.nBits)
		if got != tc.want {
			t.Errorf("readBitsFromBytes(data, %d, %d) = %d (0x%X), want %d (0x%X)",
				tc.bitPos, tc.nBits, got, got, tc.want, tc.want)
		}
	}
}

func TestSignExtend(t *testing.T) {
	cases := []struct {
		value    int16
		bitDepth int
		want     int16
	}{
		{0, 1, 0},             // 0 stays 0
		{1, 1, -1},            // 1-bit value 1 extends to -1
		{0b0111, 4, 7},        // positive 4-bit value
		{0b1000, 4, -8},       // negative 4-bit value (sign bit set)
		{0b1111, 4, -1},       // -1 in 4-bit
		{0b01111111, 8, 127},  // positive 8-bit
		{0b10000000, 8, -128}, // negative 8-bit
		{0x7FFF, 16, 32767},   // max positive 16-bit
		{-1, 16, -1},          // -1 16-bit stays -1
	}

	for _, tc := range cases {
		got := signExtend(tc.value, tc.bitDepth)
		if got != tc.want {
			t.Errorf("signExtend(%d, %d) = %d, want %d", tc.value, tc.bitDepth, got, tc.want)
		}
	}
}

func TestIntegrate(t *testing.T) {
	cases := []struct {
		name  string
		input []int16
		want  []int16
	}{
		{"empty", []int16{}, []int16{}},
		{"single", []int16{5}, []int16{5}},
		{"two elements", []int16{5, 3}, []int16{5, 8}},
		{"negative delta", []int16{10, -3, -2}, []int16{10, 7, 5}},
		{"alternating", []int16{0, 1, -1, 1, -1}, []int16{0, 1, 0, 1, 0}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := integrate(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("integrate(%v) len = %d, want %d", tc.input, len(got), len(tc.want))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("integrate(%v)[%d] = %d, want %d", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestIntegrateOverflow(t *testing.T) {
	// Test overflow wrapping behavior
	input := []int16{32767, 1}
	want := []int16{32767, -32768} // 32767 + 1 wraps to -32768

	got := integrate(input)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("integrate(%v)[%d] = %d, want %d", input, i, got[i], want[i])
		}
	}

	// Test underflow
	input2 := []int16{-32768, -1}
	want2 := []int16{-32768, 32767} // -32768 - 1 wraps to 32767

	got2 := integrate(input2)
	for i := range want2 {
		if got2[i] != want2[i] {
			t.Errorf("integrate(%v)[%d] = %d, want %d", input2, i, got2[i], want2[i])
		}
	}
}

func TestDecompressByteArraySimple(t *testing.T) {
	// Create a simple compressed byte array
	// Group 1: header=0 (1-bit values), 8 zeros: 0b0000_0000_0000
	// 4-bit header (0) + 8 1-bit values (all 0) = 12 bits = 0x000 with padding
	compressed := []byte{0x00, 0x00}

	result := DecompressByteArray(compressed, 8, false, 8)

	if len(result) != 8 {
		t.Fatalf("expected 8 elements, got %d", len(result))
	}

	for i, v := range result {
		if v != 0 {
			t.Errorf("result[%d] = %d, want 0", i, v)
		}
	}
}

func TestDecompressByteArrayWithDifferentiation(t *testing.T) {
	// Test with differentiation (integration)
	// First create simple delta-encoded data where we know the expected result
	// Group with header=0 (1-bit values), all zeros -> all zeros after integration
	compressed := []byte{0x00, 0x00}

	result := DecompressByteArray(compressed, 8, true, 8)

	// All zeros in, all zeros after integration
	for i, v := range result {
		if v != 0 {
			t.Errorf("result[%d] = %d, want 0", i, v)
		}
	}
}

func TestBitMasksTable(t *testing.T) {
	// Verify the bit masks are correct powers of 2
	for i, mask := range bitMasks {
		expected := uint32(1) << uint(i)
		if mask != expected {
			t.Errorf("bitMasks[%d] = %d, want %d", i, mask, expected)
		}
	}
}

func TestSignExtendMasksTable(t *testing.T) {
	// Verify key sign extension masks
	if signExtendMasks[0] != 0 {
		t.Errorf("signExtendMasks[0] = %d, want 0", signExtendMasks[0])
	}
	if signExtendMasks[1] != -32768 {
		t.Errorf("signExtendMasks[1] = %d, want -32768", signExtendMasks[1])
	}
	if signExtendMasks[16] != -1 {
		t.Errorf("signExtendMasks[16] = %d, want -1", signExtendMasks[16])
	}
}
