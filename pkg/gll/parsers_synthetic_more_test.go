package gll

import (
	"bytes"
	"encoding/binary"
	"testing"

	internalgll "github.com/cwbudde/gll-tools/internal/gll"
)

// Second batch of synthetic byte-builder tests targeting the remaining
// 0–60% covered parsers in pkg/gll: parseLimit, parseWarning, parseEdge,
// parseGenSystemPreset, parseLabeledVector3D, parseGenericBaseFilterBase,
// parseFilterFunction, parseLogSpectrumFilter, parseBoxInputConfig,
// readCompressedResponseData (error branches only — happy-path requires
// BitCompression-encoded bytes which already get coverage transitively from
// SphereLine19 fixtures via real parses driven elsewhere).

// ---- parseLimit -------------------------------------------------------------

func buildLimitBlock(frame, boxType string, lt int32, value float64) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // vcheck
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // subver
	writeStr(body, frame)
	_ = binary.Write(body, binary.LittleEndian, lt)
	writeStr(body, boxType)
	_ = binary.Write(body, binary.LittleEndian, value)
	return withBlockSize(body.Bytes())
}

func TestParseLimit_HappyPath(t *testing.T) {
	raw := buildLimitBlock("frameA", "boxX", 2, 105.5)
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	limit, err := parseLimit(br)
	if err != nil {
		t.Fatalf("parseLimit: %v", err)
	}
	if limit.Frame != "frameA" || limit.BoxType != "boxX" {
		t.Errorf("got %+v, want Frame=frameA BoxType=boxX", limit)
	}
	if limit.Type != LimitType(2) {
		t.Errorf("Type = %d, want 2", limit.Type)
	}
	if limit.LimitValue != 105.5 {
		t.Errorf("LimitValue = %v, want 105.5", limit.LimitValue)
	}
}

func TestParseLimit_Errors(t *testing.T) {
	t.Run("invalid block size", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseLimit(br); err == nil {
			t.Fatal("want error on zero block size")
		}
	})
	t.Run("bad version", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(40))
		_ = binary.Write(buf, binary.LittleEndian, int16(7))
		buf.Write(bytes.Repeat([]byte{0}, 34))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseLimit(br); err == nil {
			t.Fatal("want error on bad version")
		}
	})
}

// ---- parseWarning -----------------------------------------------------------

func buildWarningBlock(frame, text string, wt int32, value float64) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	writeStr(body, frame)
	_ = binary.Write(body, binary.LittleEndian, wt)
	writeStr(body, text)
	_ = binary.Write(body, binary.LittleEndian, value)
	return withBlockSize(body.Bytes())
}

func TestParseWarning_HappyPath(t *testing.T) {
	raw := buildWarningBlock("frameA", "thermal", 3, 88.0)
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	w, err := parseWarning(br)
	if err != nil {
		t.Fatalf("parseWarning: %v", err)
	}
	if w.Frame != "frameA" || w.Text != "thermal" {
		t.Errorf("got %+v", w)
	}
	if w.Type != WarningType(3) {
		t.Errorf("Type = %d, want 3", w.Type)
	}
	if w.LimitValue != 88.0 {
		t.Errorf("LimitValue = %v, want 88", w.LimitValue)
	}
}

func TestParseWarning_BadVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int32(40))
	_ = binary.Write(buf, binary.LittleEndian, int16(99))
	buf.Write(bytes.Repeat([]byte{0}, 34))
	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	if _, err := parseWarning(br); err == nil {
		t.Fatal("want error on bad version")
	}
}

// ---- parseEdge --------------------------------------------------------------

func buildEdgeBlock(color, v1, v2 int32, label string, twin bool) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	_ = binary.Write(body, binary.LittleEndian, color)
	_ = binary.Write(body, binary.LittleEndian, v1)
	_ = binary.Write(body, binary.LittleEndian, v2)
	writeStr(body, label)
	if twin {
		body.WriteByte(1)
	} else {
		body.WriteByte(0)
	}
	return withBlockSize(body.Bytes())
}

func TestParseEdge_HappyPath(t *testing.T) {
	raw := buildEdgeBlock(0xAABBCC, 5, 17, "edge-1", true)
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	e, err := parseEdge(br)
	if err != nil {
		t.Fatalf("parseEdge: %v", err)
	}
	if e.Color != 0xAABBCC || e.V1 != 5 || e.V2 != 17 || !e.HasTwin || e.Label != "edge-1" {
		t.Errorf("got %+v", e)
	}
}

func TestParseEdge_BadVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int32(20))
	_ = binary.Write(buf, binary.LittleEndian, int16(13))
	buf.Write(bytes.Repeat([]byte{0}, 14))
	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	if _, err := parseEdge(br); err == nil {
		t.Fatal("want error on bad version")
	}
}

// ---- parseGenSystemPreset ---------------------------------------------------

func buildPresetBlock(label, key string, configBytes []byte) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	writeStr(body, label)
	writeStr(body, key)
	body.Write(configBytes)
	return withBlockSize(body.Bytes())
}

func TestParseGenSystemPreset_WithConfig(t *testing.T) {
	cfg := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02}
	raw := buildPresetBlock("Preset-1", "p1", cfg)
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	p, err := parseGenSystemPreset(br)
	if err != nil {
		t.Fatalf("parseGenSystemPreset: %v", err)
	}
	if p.Label != "Preset-1" || p.Key != "p1" {
		t.Errorf("Label/Key = %q/%q", p.Label, p.Key)
	}
	if p.ConfigSize != len(cfg) {
		t.Errorf("ConfigSize = %d, want %d", p.ConfigSize, len(cfg))
	}
	if !bytes.Equal(p.ConfigRaw, cfg) {
		t.Errorf("ConfigRaw = %X, want %X", p.ConfigRaw, cfg)
	}
}

func TestParseGenSystemPreset_NoConfig(t *testing.T) {
	raw := buildPresetBlock("Preset-2", "p2", nil)
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	p, err := parseGenSystemPreset(br)
	if err != nil {
		t.Fatalf("parseGenSystemPreset: %v", err)
	}
	if p.ConfigSize != 0 || p.ConfigRaw != nil {
		t.Errorf("expected no config, got size=%d raw=%v", p.ConfigSize, p.ConfigRaw)
	}
}

func TestParseGenSystemPreset_BadVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int32(20))
	_ = binary.Write(buf, binary.LittleEndian, int16(11))
	buf.Write(bytes.Repeat([]byte{0}, 14))
	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	if _, err := parseGenSystemPreset(br); err == nil {
		t.Fatal("want error on bad version")
	}
}

// ---- parseLabeledVector3D ---------------------------------------------------

func buildLabeledVector3DBlock(label string, subVer int16, x, y, z float64) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // vcheck
	_ = binary.Write(body, binary.LittleEndian, subVer)   // subver
	writeStr(body, label)
	if subVer == 0 {
		_ = binary.Write(body, binary.LittleEndian, int32(0)) // padding
		_ = binary.Write(body, binary.LittleEndian, int32(0)) // padding
	}
	_ = binary.Write(body, binary.LittleEndian, x)
	_ = binary.Write(body, binary.LittleEndian, y)
	_ = binary.Write(body, binary.LittleEndian, z)
	return withBlockSize(body.Bytes())
}

func TestParseLabeledVector3D_SubVer1(t *testing.T) {
	raw := buildLabeledVector3DBlock("ear", 1, 1.0, 2.0, 3.0)
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	lv, err := parseLabeledVector3D(br)
	if err != nil {
		t.Fatalf("parseLabeledVector3D: %v", err)
	}
	if lv.Label != "ear" {
		t.Errorf("Label = %q, want ear", lv.Label)
	}
	if lv.Vector.X != 1.0 || lv.Vector.Y != 2.0 || lv.Vector.Z != 3.0 {
		t.Errorf("Vector = %+v", lv.Vector)
	}
}

func TestParseLabeledVector3D_SubVer0SkipsPadding(t *testing.T) {
	raw := buildLabeledVector3DBlock("nose", 0, 0.5, -0.5, 1.5)
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	lv, err := parseLabeledVector3D(br)
	if err != nil {
		t.Fatalf("parseLabeledVector3D: %v", err)
	}
	if lv.Vector.X != 0.5 || lv.Vector.Y != -0.5 || lv.Vector.Z != 1.5 {
		t.Errorf("Vector = %+v", lv.Vector)
	}
}

func TestParseLabeledVector3D_BadVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int32(40))
	_ = binary.Write(buf, binary.LittleEndian, int16(9))
	buf.Write(bytes.Repeat([]byte{0}, 34))
	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	if _, err := parseLabeledVector3D(br); err == nil {
		t.Fatal("want error on bad version")
	}
}

// ---- parseGenericBaseFilterBase --------------------------------------------

func buildGenericBaseFilterBase(bypass, invert bool, gain, delay float64, label, key string) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // vcheck
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // subver
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // reserved
	if bypass {
		_ = binary.Write(body, binary.LittleEndian, int32(1))
	} else {
		_ = binary.Write(body, binary.LittleEndian, int32(0))
	}
	if invert {
		_ = binary.Write(body, binary.LittleEndian, int32(1))
	} else {
		_ = binary.Write(body, binary.LittleEndian, int32(0))
	}
	_ = binary.Write(body, binary.LittleEndian, gain)
	_ = binary.Write(body, binary.LittleEndian, delay)
	writeStr(body, label)
	writeStr(body, key)
	return withBlockSize(body.Bytes())
}

func TestParseGenericBaseFilterBase_HappyPath(t *testing.T) {
	raw := buildGenericBaseFilterBase(true, false, 6.0, 0.001, "EQ-Lo", "k1")
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	f, err := parseGenericBaseFilterBase(br, int64(len(raw)))
	if err != nil {
		t.Fatalf("parseGenericBaseFilterBase: %v", err)
	}
	if !f.ByPass {
		t.Error("ByPass = false, want true")
	}
	if f.InvertPolarity {
		t.Error("InvertPolarity = true, want false")
	}
	if f.Gain != 6.0 {
		t.Errorf("Gain = %v, want 6", f.Gain)
	}
	if f.Delay != 0.001 {
		t.Errorf("Delay = %v, want 0.001", f.Delay)
	}
	if f.Label != "EQ-Lo" || f.Key != "k1" {
		t.Errorf("Label/Key = %q/%q", f.Label, f.Key)
	}
}

func TestParseGenericBaseFilterBase_EarlyReturns(t *testing.T) {
	t.Run("offset already at maxOffset", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader([]byte{0, 0}))
		_, _ = br.ReadInt16()
		f, err := parseGenericBaseFilterBase(br, 0)
		if err != nil || f == nil {
			t.Errorf("got (%v, %v), want non-nil filter and nil err", f, err)
		}
	})
	t.Run("zero block size", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		f, err := parseGenericBaseFilterBase(br, 100)
		if err != nil || f == nil {
			t.Errorf("got (%v, %v), want non-nil filter and nil err", f, err)
		}
	})
	t.Run("bad version", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(40))
		_ = binary.Write(buf, binary.LittleEndian, int16(7))
		buf.Write(bytes.Repeat([]byte{0}, 34))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		f, _ := parseGenericBaseFilterBase(br, 100)
		// Should return a filter struct (may be zero) without erroring.
		if f == nil {
			t.Error("want non-nil filter on bad version")
		}
	})
}

// ---- parseFilterFunction ----------------------------------------------------

func buildFilterFunctionBlock(filterType, filterShape, order int32, freqCrit float64, align int32, qFactor, gainDB float64) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // vcheck
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // subver
	_ = binary.Write(body, binary.LittleEndian, filterType)
	_ = binary.Write(body, binary.LittleEndian, filterShape)
	_ = binary.Write(body, binary.LittleEndian, order)
	_ = binary.Write(body, binary.LittleEndian, freqCrit)
	_ = binary.Write(body, binary.LittleEndian, align)
	_ = binary.Write(body, binary.LittleEndian, float64(0)) // reserved
	_ = binary.Write(body, binary.LittleEndian, qFactor)
	_ = binary.Write(body, binary.LittleEndian, gainDB)
	return withBlockSize(body.Bytes())
}

func TestParseFilterFunction_HappyPath(t *testing.T) {
	raw := buildFilterFunctionBlock(
		int32(FilterTypePeak),
		int32(FilterShapeBessel),
		2,
		1000.0,
		int32(FilterAlignLevel3dB),
		0.707,
		3.5,
	)
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	p, err := parseFilterFunction(br, int64(len(raw)))
	if err != nil {
		t.Fatalf("parseFilterFunction: %v", err)
	}
	if p.FilterType != FilterTypePeak {
		t.Errorf("FilterType = %v, want Peak", p.FilterType)
	}
	if p.FilterShape != FilterShapeBessel {
		t.Errorf("FilterShape = %v, want Bessel", p.FilterShape)
	}
	if p.Order != 2 {
		t.Errorf("Order = %d, want 2", p.Order)
	}
	if p.FreqCritInHz != 1000.0 {
		t.Errorf("FreqCritInHz = %v, want 1000", p.FreqCritInHz)
	}
	if p.QFactor != 0.707 {
		t.Errorf("QFactor = %v, want 0.707", p.QFactor)
	}
	if p.ParametricGainIndB != 3.5 {
		t.Errorf("Gain = %v, want 3.5", p.ParametricGainIndB)
	}
}

func TestParseFilterFunction_EarlyReturns(t *testing.T) {
	t.Run("offset >= maxOffset", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader([]byte{0, 0}))
		_, _ = br.ReadInt16()
		p, err := parseFilterFunction(br, 0)
		if p != nil || err != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", p, err)
		}
	})
	t.Run("zero block size", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		p, err := parseFilterFunction(br, 100)
		if p != nil || err != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", p, err)
		}
	})
	t.Run("bad version", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(40))
		_ = binary.Write(buf, binary.LittleEndian, int16(7))
		buf.Write(bytes.Repeat([]byte{0}, 34))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseFilterFunction(br, 100); err == nil {
			t.Fatal("want error on bad version")
		}
	})
}

// ---- parseLogSpectrumFilter -------------------------------------------------

// buildLogSpectrumFilterBlockVCheck1 builds a vcheck=1 LogSpectrumFilter
// (TransferFunctionLsPs format) by composing a base block + a TFLP block.
func buildLogSpectrumFilterBlockVCheck1() []byte {
	base := buildGenericBaseFilterBase(false, false, 0, 0, "L1", "k")
	tflp := buildFilterTransferFunctionLP(24, 22.1, []int16{100, 200}, []int16{10, 20}, 0)

	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(1)) // vcheck=1 → TFLP path
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // subver
	body.Write(base)
	body.Write(tflp)
	return withBlockSize(body.Bytes())
}

func TestParseLogSpectrumFilter_VCheck1_TFLP(t *testing.T) {
	raw := buildLogSpectrumFilterBlockVCheck1()
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	f, err := parseLogSpectrumFilter(br, int64(len(raw)))
	if err != nil {
		t.Fatalf("parseLogSpectrumFilter: %v", err)
	}
	if f.Label != "L1" || f.Key != "k" {
		t.Errorf("base fields not parsed: %+v", f)
	}
	if f.LogSpectrum == nil {
		t.Fatal("LogSpectrum should be populated by TFLP path")
	}
	if f.LogSpectrum.BandsPerOctave != 24 {
		t.Errorf("LogSpectrum.BandsPerOctave = %d, want 24", f.LogSpectrum.BandsPerOctave)
	}
}

func TestParseLogSpectrumFilter_EarlyReturns(t *testing.T) {
	t.Run("offset >= maxOffset", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader([]byte{0, 0}))
		_, _ = br.ReadInt16()
		f, err := parseLogSpectrumFilter(br, 0)
		if f != nil || err != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", f, err)
		}
	})
	t.Run("unsupported version", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(40))
		_ = binary.Write(buf, binary.LittleEndian, int16(7)) // > 1
		buf.Write(bytes.Repeat([]byte{0}, 34))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseLogSpectrumFilter(br, 100); err == nil {
			t.Fatal("want error on bad version")
		}
	})
}

// ---- parseBoxInputConfig ----------------------------------------------------

func buildBoxInputConfigBlock(label, key string, inputs [][]byte) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // vcheck
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // subver
	writeStr(body, label)
	writeStr(body, key)
	body.Write(buildBoxInputBufferBlock(inputs))
	return withBlockSize(body.Bytes())
}

func TestParseBoxInputConfig_HappyPath(t *testing.T) {
	inputs := [][]byte{
		buildBoxInputBlock("LF", nil, 0, 0),
		buildBoxInputBlock("HF", []SourceFilterLink{{SourceKey: "s", FilterGrpKey: "f"}}, 8, 1),
	}
	raw := buildBoxInputConfigBlock("Bi-amp", "ic-1", inputs)

	br := internalgll.NewByteReader(bytes.NewReader(raw))
	cfg, err := parseBoxInputConfig(br, int64(len(raw)))
	if err != nil {
		t.Fatalf("parseBoxInputConfig: %v", err)
	}
	if cfg.Label != "Bi-amp" || cfg.Key != "ic-1" {
		t.Errorf("Label/Key = %q/%q", cfg.Label, cfg.Key)
	}
	if len(cfg.Inputs) != 2 {
		t.Fatalf("Inputs len = %d, want 2", len(cfg.Inputs))
	}
}

func TestParseBoxInputConfig_BadVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int32(40))
	_ = binary.Write(buf, binary.LittleEndian, int16(9))
	buf.Write(bytes.Repeat([]byte{0}, 34))
	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	if _, err := parseBoxInputConfig(br, 100); err == nil {
		t.Fatal("want error on bad version")
	}
}

func TestParseBoxInputConfig_ZeroBlockSize(t *testing.T) {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int32(0))
	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	_, err := parseBoxInputConfig(br, 100)
	if err == nil {
		t.Fatal("want error on zero block size")
	}
}

// ---- readCompressedResponseData (error branches only) ----------------------

func TestReadCompressedResponseData_Errors(t *testing.T) {
	t.Run("level count read fails", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader(nil))
		levels, phases := readCompressedResponseData(br, 100, 0)
		if levels != nil || phases != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", levels, phases)
		}
	})
	t.Run("compressed length out of range", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(10))     // levelCount
		_ = binary.Write(buf, binary.LittleEndian, int32(999999)) // levelCompressedLen way too big
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		levels, phases := readCompressedResponseData(br, 10, 1024)
		if levels != nil || phases != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", levels, phases)
		}
	})
	t.Run("negative compressed length", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(10))
		_ = binary.Write(buf, binary.LittleEndian, int32(-1))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		levels, phases := readCompressedResponseData(br, 10, 1024)
		if levels != nil || phases != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", levels, phases)
		}
	})
	t.Run("truncated level bytes", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(5)) // levelCount
		_ = binary.Write(buf, binary.LittleEndian, int32(2)) // 4 bytes claimed
		buf.Write([]byte{0x01})                              // only 1 byte provided
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		levels, _ := readCompressedResponseData(br, 10, 1024)
		if levels != nil {
			t.Errorf("levels = %v, want nil (truncated read)", levels)
		}
	})
}
