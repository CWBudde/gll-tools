package gll

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"

	internalgll "github.com/cwbudde/gll-tools/internal/gll"
)

// ---- Byte-builder helpers ----

func wI16(b *bytes.Buffer, v int16) {
	_ = binary.Write(b, binary.LittleEndian, v)
}

func wI32(b *bytes.Buffer, v int32) {
	_ = binary.Write(b, binary.LittleEndian, v)
}

func wF64(b *bytes.Buffer, v float64) {
	_ = binary.Write(b, binary.LittleEndian, v)
}

func wStr(b *bytes.Buffer, s string) {
	if len(s) > math.MaxInt16 {
		panic("test string too long for int16 length prefix")
	}
	wI16(b, int16(len(s))) //nolint:gosec // bounded above
	b.WriteString(s)
}

// wrapBlock prefixes inner with an int32 size that includes the size field.
func wrapBlock(inner []byte) []byte {
	out := new(bytes.Buffer)
	if len(inner)+4 > math.MaxInt32 {
		panic("test block too large for int32 size prefix")
	}
	wI32(out, int32(len(inner)+4)) //nolint:gosec // bounded above
	out.Write(inner)
	return out.Bytes()
}

// emptyElementBuffer builds a GenSystemConfig element buffer holding zero elements.
func emptyElementBuffer(sver int16) []byte {
	inner := new(bytes.Buffer)
	wI16(inner, 0)    // vcheck
	wI16(inner, sver) // sver
	wI32(inner, 0)    // count
	return wrapBlock(inner.Bytes())
}

// configBytesV0 builds a minimal valid GenSystemConfig (sver=0, no elements).
func configBytesV0(grid float64, frameKey, clusterKey string) []byte {
	inner := new(bytes.Buffer)
	wI16(inner, 0) // vcheck
	wI16(inner, 0) // sver
	wF64(inner, grid)
	wI32(inner, 7) // unknown_int32
	wStr(inner, frameKey)
	wStr(inner, clusterKey)
	inner.Write(emptyElementBuffer(0))
	return wrapBlock(inner.Bytes())
}

// configBytesV1 builds sver=1 with an explicit system type. If systemType==1
// and clusterKey is empty, a user cluster setup block is appended.
func configBytesV1(systemType int32, clusterKey string, userClusterPayload []byte) []byte {
	inner := new(bytes.Buffer)
	wI16(inner, 0) // vcheck
	wI16(inner, 1) // sver
	wF64(inner, 0.0)
	wI32(inner, 0)
	wStr(inner, "frameX")
	wStr(inner, clusterKey)
	inner.Write(emptyElementBuffer(0))
	wI32(inner, systemType)
	if systemType == 1 && clusterKey == "" {
		inner.Write(wrapBlock(userClusterPayload))
	}
	return wrapBlock(inner.Bytes())
}

// elementBytesNoSources builds a single element block with sources=0.
func elementBytesNoSources(sver int16, boxKey, inputKey string, splay, gain float64) []byte {
	inner := new(bytes.Buffer)
	wI16(inner, 0)    // vcheck
	wI16(inner, sver) // sver
	wStr(inner, boxKey)
	wF64(inner, splay)
	wF64(inner, gain)
	wStr(inner, inputKey)
	wI32(inner, 0) // sources=0 — skips per-source loop and override loop
	return wrapBlock(inner.Bytes())
}

// elementsBuffer wraps a list of element blocks in an element-buffer container.
func elementsBuffer(elemSver int16, elements ...[]byte) []byte {
	inner := new(bytes.Buffer)
	wI16(inner, 0)        // vcheck
	wI16(inner, elemSver) // sver
	if len(elements) > math.MaxInt32 {
		panic("too many elements for int32 count")
	}
	wI32(inner, int32(len(elements))) //nolint:gosec // bounded above
	for _, e := range elements {
		inner.Write(e)
	}
	return wrapBlock(inner.Bytes())
}

// ---- DecodeGenSystemConfigRaw ----

func TestDecodeGenSystemConfigRaw_SubVersion0(t *testing.T) {
	data := configBytesV0(2.5, "frameA", "clusterA")
	cfg, err := DecodeGenSystemConfigRaw(data)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if cfg.VersionCheck != 0 {
		t.Errorf("VersionCheck = %d, want 0", cfg.VersionCheck)
	}
	if cfg.SubVersion != 0 {
		t.Errorf("SubVersion = %d, want 0", cfg.SubVersion)
	}
	if cfg.GridAngle != 2.5 {
		t.Errorf("GridAngle = %v, want 2.5", cfg.GridAngle)
	}
	if cfg.UnknownInt32 != 7 {
		t.Errorf("UnknownInt32 = %d, want 7", cfg.UnknownInt32)
	}
	if cfg.FrameKey != "frameA" {
		t.Errorf("FrameKey = %q, want %q", cfg.FrameKey, "frameA")
	}
	if cfg.ClusterSetupKey != "clusterA" {
		t.Errorf("ClusterSetupKey = %q, want %q", cfg.ClusterSetupKey, "clusterA")
	}
	if len(cfg.Elements) != 0 {
		t.Errorf("Elements length = %d, want 0", len(cfg.Elements))
	}
	// SystemType is only read for sver>=1
	if cfg.SystemType != 0 {
		t.Errorf("SystemType = %d, want 0 (unset for sver=0)", cfg.SystemType)
	}
	if cfg.UserClusterSetupPresent {
		t.Errorf("UserClusterSetupPresent = true, want false")
	}
}

func TestDecodeGenSystemConfigRaw_SubVersion1ReadsSystemType(t *testing.T) {
	data := configBytesV1(2, "namedCluster", nil)
	cfg, err := DecodeGenSystemConfigRaw(data)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if cfg.SubVersion != 1 {
		t.Errorf("SubVersion = %d, want 1", cfg.SubVersion)
	}
	if cfg.SystemType != 2 {
		t.Errorf("SystemType = %d, want 2", cfg.SystemType)
	}
	if cfg.UserClusterSetupPresent {
		t.Errorf("UserClusterSetupPresent = true, want false (named cluster key set)")
	}
}

func TestDecodeGenSystemConfigRaw_UserClusterSetupBlock(t *testing.T) {
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02} // arbitrary user cluster bytes
	data := configBytesV1(1, "", payload)
	cfg, err := DecodeGenSystemConfigRaw(data)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !cfg.UserClusterSetupPresent {
		t.Fatal("UserClusterSetupPresent = false, want true")
	}
	// UserClusterSetupSize is the wrapped block size (payload + 4-byte size field).
	wantSize := len(payload) + 4
	if cfg.UserClusterSetupSize != wantSize {
		t.Errorf("UserClusterSetupSize = %d, want %d", cfg.UserClusterSetupSize, wantSize)
	}
}

func TestDecodeGenSystemConfigRaw_NegativeSubVersionSkipsBody(t *testing.T) {
	// sver < 0 means none of the metadata fields are read. Build a config block
	// containing only blockSize + vcheck + sver and trailing slack.
	inner := new(bytes.Buffer)
	wI16(inner, 0)                 // vcheck
	wI16(inner, -1)                // sver
	inner.Write([]byte{0xFF, 0x0}) // unread slack inside the block
	data := wrapBlock(inner.Bytes())

	cfg, err := DecodeGenSystemConfigRaw(data)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if cfg.SubVersion != -1 {
		t.Errorf("SubVersion = %d, want -1", cfg.SubVersion)
	}
	if cfg.GridAngle != 0 || cfg.FrameKey != "" || cfg.ClusterSetupKey != "" {
		t.Errorf("body fields should be zero-valued for sver < 0, got %+v", cfg)
	}
}

func TestDecodeGenSystemConfigRaw_MultipleElements(t *testing.T) {
	e1 := elementBytesNoSources(0, "boxA", "inputA", 0.1, 1.0)
	e2 := elementBytesNoSources(1, "boxB", "inputB", math.Pi, -3.5)
	elemBuf := elementsBuffer(1, e1, e2)

	inner := new(bytes.Buffer)
	wI16(inner, 0) // vcheck
	wI16(inner, 0) // sver=0 → no SystemType read
	wF64(inner, 0)
	wI32(inner, 0)
	wStr(inner, "")
	wStr(inner, "")
	inner.Write(elemBuf)
	data := wrapBlock(inner.Bytes())

	cfg, err := DecodeGenSystemConfigRaw(data)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got, want := len(cfg.Elements), 2; got != want {
		t.Fatalf("Elements length = %d, want %d", got, want)
	}
	if cfg.Elements[0].BoxTypeKey != "boxA" || cfg.Elements[1].BoxTypeKey != "boxB" {
		t.Errorf("BoxTypeKeys = [%q, %q], want [boxA, boxB]",
			cfg.Elements[0].BoxTypeKey, cfg.Elements[1].BoxTypeKey)
	}
	if cfg.Elements[1].SplayAngle != math.Pi {
		t.Errorf("Elements[1].SplayAngle = %v, want %v", cfg.Elements[1].SplayAngle, math.Pi)
	}
	if cfg.Elements[1].Gain != -3.5 {
		t.Errorf("Elements[1].Gain = %v, want -3.5", cfg.Elements[1].Gain)
	}
	for i, e := range cfg.Elements {
		if e.Sources != 0 {
			t.Errorf("Elements[%d].Sources = %d, want 0", i, e.Sources)
		}
	}
}

// ---- Error paths ----

func TestDecodeGenSystemConfigRaw_Errors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantSub string
	}{
		{
			name:    "empty input",
			data:    nil,
			wantSub: "reading block size",
		},
		{
			name: "zero block size",
			data: func() []byte {
				b := new(bytes.Buffer)
				wI32(b, 0)
				return b.Bytes()
			}(),
			wantSub: "invalid block size",
		},
		{
			name: "negative block size",
			data: func() []byte {
				b := new(bytes.Buffer)
				wI32(b, -1)
				return b.Bytes()
			}(),
			wantSub: "invalid block size",
		},
		{
			name: "unsupported vcheck",
			data: func() []byte {
				inner := new(bytes.Buffer)
				wI16(inner, 1) // vcheck != 0
				wI16(inner, 0)
				return wrapBlock(inner.Bytes())
			}(),
			wantSub: "unsupported vcheck",
		},
		{
			name: "truncated after vcheck",
			data: func() []byte {
				// blockSize claims plenty of bytes, but only vcheck is present.
				out := new(bytes.Buffer)
				wI32(out, 32)
				wI16(out, 0)
				return out.Bytes()
			}(),
			wantSub: "reading sver",
		},
		{
			name: "truncated grid angle (sver=0, missing body)",
			data: func() []byte {
				inner := new(bytes.Buffer)
				wI16(inner, 0)
				wI16(inner, 0)
				return wrapBlock(inner.Bytes())
			}(),
			wantSub: "reading grid angle",
		},
		{
			name: "invalid element buffer count",
			data: func() []byte {
				badElemBuf := func() []byte {
					inner := new(bytes.Buffer)
					wI16(inner, 0)
					wI16(inner, 0)
					wI32(inner, -5) // negative count
					return wrapBlock(inner.Bytes())
				}()
				inner := new(bytes.Buffer)
				wI16(inner, 0)
				wI16(inner, 0)
				wF64(inner, 0)
				wI32(inner, 0)
				wStr(inner, "")
				wStr(inner, "")
				inner.Write(badElemBuf)
				return wrapBlock(inner.Bytes())
			}(),
			wantSub: "invalid element count",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeGenSystemConfigRaw(tc.data)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestDecodeGenSystemConfigRaw_ElementErrors(t *testing.T) {
	// Element with vcheck != 0 propagates as "parsing element 0: ..."
	badElement := func() []byte {
		inner := new(bytes.Buffer)
		wI16(inner, 9) // vcheck != 0
		wI16(inner, 0)
		return wrapBlock(inner.Bytes())
	}()
	elemBuf := elementsBuffer(0, badElement)
	inner := new(bytes.Buffer)
	wI16(inner, 0)
	wI16(inner, 0)
	wF64(inner, 0)
	wI32(inner, 0)
	wStr(inner, "")
	wStr(inner, "")
	inner.Write(elemBuf)
	data := wrapBlock(inner.Bytes())

	_, err := DecodeGenSystemConfigRaw(data)
	if err == nil {
		t.Fatal("expected error for element with vcheck != 0")
	}
	if !strings.Contains(err.Error(), "parsing element 0") {
		t.Errorf("error = %q, want substring %q", err.Error(), "parsing element 0")
	}
	if !strings.Contains(err.Error(), "unsupported element vcheck") {
		t.Errorf("error = %q, want substring %q", err.Error(), "unsupported element vcheck")
	}
}

// ---- skipBlockRaw ----

func TestSkipBlockRaw(t *testing.T) {
	t.Run("skips block and returns size", func(t *testing.T) {
		buf := new(bytes.Buffer)
		wI32(buf, 16) // block size including the size field
		buf.Write(bytes.Repeat([]byte{0xAA}, 12))
		// Trailing sentinel byte that should remain available after skipping.
		buf.WriteByte(0x55)

		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		size, err := skipBlockRaw(br)
		if err != nil {
			t.Fatalf("skipBlockRaw error: %v", err)
		}
		if size != 16 {
			t.Errorf("size = %d, want 16", size)
		}
		// Verify position: next byte read should be the sentinel.
		next, err := br.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte after skip: %v", err)
		}
		if next != 0x55 {
			t.Errorf("byte after skip = 0x%X, want 0x55", next)
		}
	})

	t.Run("rejects zero block size", func(t *testing.T) {
		buf := new(bytes.Buffer)
		wI32(buf, 0)
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		_, err := skipBlockRaw(br)
		if err == nil || !strings.Contains(err.Error(), "invalid block size") {
			t.Errorf("err = %v, want 'invalid block size'", err)
		}
	})

	t.Run("propagates short read on size", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader(nil))
		_, err := skipBlockRaw(br)
		if err == nil {
			t.Error("expected error on empty input")
		}
	})
}

// ---- SplayAngleDeg ----

func TestSplayAngleDeg(t *testing.T) {
	tests := []struct {
		name string
		rad  float64
		deg  float64
	}{
		{"zero", 0, 0},
		{"pi → 180", math.Pi, 180},
		{"pi/2 → 90", math.Pi / 2, 90},
		{"-pi → -180", -math.Pi, -180},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GenSystemConfigElement{SplayAngle: tc.rad}.SplayAngleDeg()
			if math.Abs(got-tc.deg) > 1e-9 {
				t.Errorf("SplayAngleDeg(%v rad) = %v, want %v", tc.rad, got, tc.deg)
			}
		})
	}
}

// ---- Round-trip / integration with real preset data ----

// TestDecodeGenSystemConfigRaw_FromRealPresets walks every preset across a set
// of sample files. Where ConfigRaw is present, it confirms the decoder accepts
// the bytes and produces self-consistent metadata.
func TestDecodeGenSystemConfigRaw_FromRealPresets(t *testing.T) {
	files := []string{
		"TiRAY-V1_3.gll",
		"N-RAY-V0_3 Beta.gll",
		"Coda-Audio G-Series-V1_2.gll",
		"LX-10 ASX_gll.gll",
		"LX-20 ASX_gll.gll",
		"D12-v10.gll",
		"APS-V1_1.gll",
	}

	totalDecoded := 0
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			gllFile := parseTestFile(t, name)
			if gllFile.Database == nil {
				t.Skip("no database")
			}
			if len(gllFile.Database.Presets) == 0 {
				t.Skip("no presets")
			}

			decodedHere := 0
			for i, preset := range gllFile.Database.Presets {
				if len(preset.ConfigRaw) == 0 {
					continue
				}
				cfg, err := DecodeGenSystemConfigRaw(preset.ConfigRaw)
				if err != nil {
					t.Errorf("Presets[%d] (%q) decode error: %v", i, preset.Label, err)
					continue
				}
				if cfg.VersionCheck != 0 {
					t.Errorf("Presets[%d].VersionCheck = %d, want 0", i, cfg.VersionCheck)
				}
				if cfg.SubVersion < -1 || cfg.SubVersion > 10 {
					t.Errorf("Presets[%d].SubVersion = %d outside plausible range", i, cfg.SubVersion)
				}
				// Element box keys should be non-empty when sources>0.
				for j, el := range cfg.Elements {
					if el.Sources > 0 && el.BoxTypeKey == "" {
						t.Errorf("Presets[%d].Elements[%d] has Sources>0 but empty BoxTypeKey", i, j)
					}
					if math.IsNaN(el.SplayAngle) || math.IsInf(el.SplayAngle, 0) {
						t.Errorf("Presets[%d].Elements[%d].SplayAngle not finite: %v", i, j, el.SplayAngle)
					}
					// Override arrays only populated when element sver>=1.
					if len(el.OverrideFlags) != 0 &&
						(len(el.OverrideFlags) != len(el.OverrideLabels) ||
							len(el.OverrideFlags) != len(el.OverrideFilters)) {
						t.Errorf("Presets[%d].Elements[%d] override arrays length mismatch: flags=%d labels=%d filters=%d",
							i, j, len(el.OverrideFlags), len(el.OverrideLabels), len(el.OverrideFilters))
					}
				}
				decodedHere++
			}
			totalDecoded += decodedHere
			t.Logf("decoded %d/%d presets in %s", decodedHere, len(gllFile.Database.Presets), name)
		})
	}

	if totalDecoded == 0 {
		t.Skip("no real preset configs available; synthetic tests still validate the decoder")
	}
}
