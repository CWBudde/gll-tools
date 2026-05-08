package gll

import (
	"bytes"
	"encoding/binary"
	"testing"

	internalgll "github.com/cwbudde/gll-tools/internal/gll"
)

// These tests target five 0%-coverage parsers whose corresponding format
// features (faces, box-input configs, the newer vcheck=1 transfer-function
// dispatch path, uncompressed response data) appear in zero of the 39 .gll
// fixtures we have. Synthetic byte builders are the only option.

// writeStr writes a length-prefixed (int16) string the same way the parser
// reads it via ReadString.
func writeStr(buf *bytes.Buffer, s string) {
	_ = binary.Write(buf, binary.LittleEndian, int16(len(s))) //nolint:gosec // bounded by literal len in tests
	buf.WriteString(s)
}

// withBlockSize prepends an int32 blockSize equal to len(content) + 4 (the
// size field itself). Returns the resulting bytes.
func withBlockSize(content []byte) []byte {
	out := new(bytes.Buffer)
	_ = binary.Write(out, binary.LittleEndian, int32(len(content)+4)) //nolint:gosec
	out.Write(content)
	return out.Bytes()
}

// buildFaceBlock constructs a valid Face block matching parseFace's expected
// layout: vcheck(0) + subver + hasTwin(byte) + count(int32) + vertices(int32) +
// color(int32) + label(string).
func buildFaceBlock(twin bool, vertices []int32, color int32, label string) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // vcheck
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // subver
	if twin {
		body.WriteByte(1)
	} else {
		body.WriteByte(0)
	}
	_ = binary.Write(body, binary.LittleEndian, int32(len(vertices))) //nolint:gosec
	for _, v := range vertices {
		_ = binary.Write(body, binary.LittleEndian, v)
	}
	_ = binary.Write(body, binary.LittleEndian, color)
	writeStr(body, label)
	return withBlockSize(body.Bytes())
}

func TestParseFace_HappyPath(t *testing.T) {
	verts := []int32{1, 2, 3, 4}
	raw := buildFaceBlock(true, verts, 0xFF7733, "front")

	br := internalgll.NewByteReader(bytes.NewReader(raw))
	face, err := parseFace(br)
	if err != nil {
		t.Fatalf("parseFace: %v", err)
	}
	if !face.HasTwin {
		t.Errorf("HasTwin = false, want true")
	}
	if len(face.Vertices) != len(verts) {
		t.Fatalf("Vertices len = %d, want %d", len(face.Vertices), len(verts))
	}
	for i, v := range verts {
		if face.Vertices[i] != v {
			t.Errorf("Vertices[%d] = %d, want %d", i, face.Vertices[i], v)
		}
	}
	if face.Color != 0xFF7733 {
		t.Errorf("Color = 0x%X, want 0xFF7733", face.Color)
	}
	if face.Label != "front" {
		t.Errorf("Label = %q, want %q", face.Label, "front")
	}
	if br.Offset() != int64(len(raw)) {
		t.Errorf("consumed %d bytes, want %d", br.Offset(), len(raw))
	}
}

func TestParseFace_Errors(t *testing.T) {
	t.Run("invalid block size", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseFace(br); err == nil {
			t.Fatal("want error on zero block size")
		}
	})
	t.Run("unsupported version", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(20))
		_ = binary.Write(buf, binary.LittleEndian, int16(99)) // bad vcheck
		buf.Write(bytes.Repeat([]byte{0}, 14))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseFace(br); err == nil {
			t.Fatal("want error on bad version")
		}
	})
	t.Run("negative vertex count", func(t *testing.T) {
		body := new(bytes.Buffer)
		_ = binary.Write(body, binary.LittleEndian, int16(0))  // vcheck
		_ = binary.Write(body, binary.LittleEndian, int16(0))  // subver
		body.WriteByte(0)                                      // twin
		_ = binary.Write(body, binary.LittleEndian, int32(-1)) // bad count
		raw := withBlockSize(body.Bytes())
		br := internalgll.NewByteReader(bytes.NewReader(raw))
		if _, err := parseFace(br); err == nil {
			t.Fatal("want error on negative vertex count")
		}
	})
}

// buildBoxInputBlock matches parseBoxInput layout. subVersion==1 makes the
// parser also read RatedImpedance.
func buildBoxInputBlock(label string, links []SourceFilterLink, impedance float64, subVersion int16) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0))          // vcheck
	_ = binary.Write(body, binary.LittleEndian, subVersion)        // subver
	writeStr(body, label)                                          // label
	_ = binary.Write(body, binary.LittleEndian, int32(len(links))) //nolint:gosec
	for _, l := range links {
		writeStr(body, l.SourceKey)
		writeStr(body, l.FilterGrpKey)
	}
	if subVersion >= 1 {
		_ = binary.Write(body, binary.LittleEndian, impedance)
	}
	return withBlockSize(body.Bytes())
}

func TestParseBoxInput_HappyPath(t *testing.T) {
	links := []SourceFilterLink{
		{SourceKey: "src1", FilterGrpKey: "flt1"},
		{SourceKey: "src2", FilterGrpKey: "flt2"},
	}
	raw := buildBoxInputBlock("LF", links, 4.0, 1)

	br := internalgll.NewByteReader(bytes.NewReader(raw))
	in, err := parseBoxInput(br, int64(len(raw)))
	if err != nil {
		t.Fatalf("parseBoxInput: %v", err)
	}
	if in.Label != "LF" {
		t.Errorf("Label = %q, want LF", in.Label)
	}
	if len(in.SourceLinks) != 2 || in.SourceLinks[0].SourceKey != "src1" {
		t.Errorf("SourceLinks = %+v", in.SourceLinks)
	}
	if in.RatedImpedance != 4.0 {
		t.Errorf("RatedImpedance = %v, want 4.0", in.RatedImpedance)
	}
}

func TestParseBoxInput_NoImpedanceWhenSubVerZero(t *testing.T) {
	raw := buildBoxInputBlock("HF", nil, 0, 0)
	br := internalgll.NewByteReader(bytes.NewReader(raw))
	in, err := parseBoxInput(br, int64(len(raw)))
	if err != nil {
		t.Fatalf("parseBoxInput: %v", err)
	}
	if in.RatedImpedance != 0 {
		t.Errorf("RatedImpedance should be 0 for subVer=0, got %v", in.RatedImpedance)
	}
}

func TestParseBoxInput_Errors(t *testing.T) {
	t.Run("zero block size", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseBoxInput(br, 100); err == nil {
			t.Fatal("want error on zero block size")
		}
	})
	t.Run("bad version", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(20))
		_ = binary.Write(buf, binary.LittleEndian, int16(7))
		buf.Write(bytes.Repeat([]byte{0}, 14))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseBoxInput(br, 100); err == nil {
			t.Fatal("want error on bad version")
		}
	})
	t.Run("negative link count", func(t *testing.T) {
		body := new(bytes.Buffer)
		_ = binary.Write(body, binary.LittleEndian, int16(0))
		_ = binary.Write(body, binary.LittleEndian, int16(0))
		writeStr(body, "x")
		_ = binary.Write(body, binary.LittleEndian, int32(-5))
		raw := withBlockSize(body.Bytes())
		br := internalgll.NewByteReader(bytes.NewReader(raw))
		if _, err := parseBoxInput(br, int64(len(raw))); err == nil {
			t.Fatal("want error on negative link count")
		}
	})
}

// buildBoxInputBufferBlock wraps a list of pre-built BoxInput blocks in the
// outer buffer layout: vcheck(0) + subver + count(int32) + items...
func buildBoxInputBufferBlock(items [][]byte) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0))          // vcheck
	_ = binary.Write(body, binary.LittleEndian, int16(0))          // subver
	_ = binary.Write(body, binary.LittleEndian, int32(len(items))) //nolint:gosec
	for _, it := range items {
		body.Write(it)
	}
	return withBlockSize(body.Bytes())
}

func TestParseBoxInputBuffer_HappyPath(t *testing.T) {
	items := [][]byte{
		buildBoxInputBlock("LF", nil, 0, 0),
		buildBoxInputBlock("HF", []SourceFilterLink{{SourceKey: "s", FilterGrpKey: "f"}}, 8, 1),
	}
	raw := buildBoxInputBufferBlock(items)

	br := internalgll.NewByteReader(bytes.NewReader(raw))
	inputs, err := parseBoxInputBuffer(br, int64(len(raw)))
	if err != nil {
		t.Fatalf("parseBoxInputBuffer: %v", err)
	}
	if len(inputs) != 2 {
		t.Fatalf("got %d inputs, want 2", len(inputs))
	}
	if inputs[0].Label != "LF" || inputs[1].Label != "HF" {
		t.Errorf("labels = %q, %q", inputs[0].Label, inputs[1].Label)
	}
	if inputs[1].RatedImpedance != 8 {
		t.Errorf("inputs[1].RatedImpedance = %v, want 8", inputs[1].RatedImpedance)
	}
}

func TestParseBoxInputBuffer_EmptyAndZero(t *testing.T) {
	t.Run("empty buffer (blockSize=0)", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		inputs, err := parseBoxInputBuffer(br, 100)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if inputs != nil {
			t.Errorf("inputs = %v, want nil", inputs)
		}
	})
	t.Run("offset already at maxOffset", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader([]byte{0, 0}))
		_, _ = br.ReadInt16()
		inputs, err := parseBoxInputBuffer(br, 0)
		if err != nil || inputs != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", inputs, err)
		}
	})
	t.Run("bad version", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(20))
		_ = binary.Write(buf, binary.LittleEndian, int16(9))
		buf.Write(bytes.Repeat([]byte{0}, 14))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseBoxInputBuffer(br, 100); err == nil {
			t.Fatal("want error on bad version")
		}
	})
	t.Run("invalid count", func(t *testing.T) {
		body := new(bytes.Buffer)
		_ = binary.Write(body, binary.LittleEndian, int16(0))
		_ = binary.Write(body, binary.LittleEndian, int16(0))
		_ = binary.Write(body, binary.LittleEndian, int32(99999))
		raw := withBlockSize(body.Bytes())
		br := internalgll.NewByteReader(bytes.NewReader(raw))
		if _, err := parseBoxInputBuffer(br, int64(len(raw))); err == nil {
			t.Fatal("want error on out-of-range count")
		}
	})
}

// buildRecord constructs a Record block (parseRecord layout) with
// uncompressed int16 values.
func buildRecord(values []int16) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0))           // vcheck
	_ = binary.Write(body, binary.LittleEndian, int16(0))           // subver
	_ = binary.Write(body, binary.LittleEndian, int32(0))           // compressionType=0
	_ = binary.Write(body, binary.LittleEndian, int32(len(values))) //nolint:gosec
	for _, v := range values {
		_ = binary.Write(body, binary.LittleEndian, v)
	}
	return withBlockSize(body.Bytes())
}

// buildComplexSequence wraps two Records (level + phase).
func buildComplexSequence(level, phase []int16) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // vcheck
	_ = binary.Write(body, binary.LittleEndian, int16(0)) // subver
	body.Write(buildRecord(level))
	body.Write(buildRecord(phase))
	return withBlockSize(body.Bytes())
}

// buildFilterTransferFunctionLP constructs the full block parsed by
// parseFilterTransferFunctionLP.
func buildFilterTransferFunctionLP(bpo int32, startFreq float64, level, phase []int16, delay float64) []byte {
	body := new(bytes.Buffer)
	_ = binary.Write(body, binary.LittleEndian, int16(0))          // vcheck
	_ = binary.Write(body, binary.LittleEndian, int16(0))          // subver
	_ = binary.Write(body, binary.LittleEndian, bpo)               // BandsPerOctave
	_ = binary.Write(body, binary.LittleEndian, startFreq)         // LowestFrequency
	_ = binary.Write(body, binary.LittleEndian, int32(len(level))) //nolint:gosec
	body.Write(buildComplexSequence(level, phase))
	_ = binary.Write(body, binary.LittleEndian, delay)
	return withBlockSize(body.Bytes())
}

func TestParseFilterTransferFunctionLP_HappyPath(t *testing.T) {
	level := []int16{100, 200, 300, 400}
	phase := []int16{10, 20, 30, 40}
	raw := buildFilterTransferFunctionLP(24, 22.1, level, phase, 0.001)

	br := internalgll.NewByteReader(bytes.NewReader(raw))
	tf, err := parseFilterTransferFunctionLP(br, int64(len(raw)))
	if err != nil {
		t.Fatalf("parseFilterTransferFunctionLP: %v", err)
	}
	if tf == nil {
		t.Fatal("returned nil spectrum")
	}
	if tf.BandsPerOctave != 24 {
		t.Errorf("BandsPerOctave = %d, want 24", tf.BandsPerOctave)
	}
	if tf.LowestFrequency != 22.1 {
		t.Errorf("LowestFrequency = %v, want 22.1", tf.LowestFrequency)
	}
	if tf.NumberOfBands != int32(len(level)) { //nolint:gosec // bounded literal in tests
		t.Errorf("NumberOfBands = %d, want %d", tf.NumberOfBands, len(level))
	}
	if len(tf.Level) != len(level) {
		t.Fatalf("Level len = %d, want %d", len(tf.Level), len(level))
	}
	// Levels are scaled by 0.01 dB.
	if tf.Level[0] != 1.0 {
		t.Errorf("Level[0] = %v, want 1.0 (= 100 * 0.01)", tf.Level[0])
	}
	// Phases are scaled by 0.001 rad.
	if tf.Phase[0] != 0.01 {
		t.Errorf("Phase[0] = %v, want 0.01 (= 10 * 0.001)", tf.Phase[0])
	}
	if tf.Delay != 0.001 {
		t.Errorf("Delay = %v, want 0.001", tf.Delay)
	}
}

func TestParseFilterTransferFunctionLP_EarlyReturns(t *testing.T) {
	t.Run("offset already at maxOffset", func(t *testing.T) {
		br := internalgll.NewByteReader(bytes.NewReader([]byte{0, 0}))
		_, _ = br.ReadInt16()
		tf, err := parseFilterTransferFunctionLP(br, 0)
		if err != nil || tf != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", tf, err)
		}
	})
	t.Run("zero block size", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		tf, err := parseFilterTransferFunctionLP(br, 100)
		if err != nil || tf != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", tf, err)
		}
	})
	t.Run("unsupported version", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(40))
		_ = binary.Write(buf, binary.LittleEndian, int16(7))
		buf.Write(bytes.Repeat([]byte{0}, 34))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		if _, err := parseFilterTransferFunctionLP(br, 100); err == nil {
			t.Fatal("want error on bad version")
		}
	})
}

func TestReadUncompressedResponseData_HappyPath(t *testing.T) {
	levels := []int16{100, -200, 300}
	phases := []int16{10, 20, 30, 40}

	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int32(len(levels))) //nolint:gosec
	for _, v := range levels {
		_ = binary.Write(buf, binary.LittleEndian, v)
	}
	_ = binary.Write(buf, binary.LittleEndian, int32(len(phases))) //nolint:gosec
	for _, v := range phases {
		_ = binary.Write(buf, binary.LittleEndian, v)
	}

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	gotLevels, gotPhases := readUncompressedResponseData(br, 10)

	if len(gotLevels) != len(levels) {
		t.Fatalf("levels len = %d, want %d", len(gotLevels), len(levels))
	}
	for i, v := range levels {
		want := float64(v) * 0.01
		if gotLevels[i] != want {
			t.Errorf("levels[%d] = %v, want %v", i, gotLevels[i], want)
		}
	}
	if len(gotPhases) != len(phases) {
		t.Fatalf("phases len = %d, want %d", len(gotPhases), len(phases))
	}
	for i, v := range phases {
		want := float64(v) * 0.001
		if gotPhases[i] != want {
			t.Errorf("phases[%d] = %v, want %v", i, gotPhases[i], want)
		}
	}
}

func TestReadUncompressedResponseData_GuardsAndShortRead(t *testing.T) {
	t.Run("count > numBands rejects level data", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(999)) // claim 999 levels
		// no actual data follows
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		levels, phases := readUncompressedResponseData(br, 10) // numBands=10
		if levels != nil {
			t.Errorf("levels = %v, want nil (count > numBands)", levels)
		}
		if phases != nil {
			t.Errorf("phases = %v, want nil", phases)
		}
	})
	t.Run("zero counts return nil slices", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		_ = binary.Write(buf, binary.LittleEndian, int32(0))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		levels, phases := readUncompressedResponseData(br, 10)
		if levels != nil || phases != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", levels, phases)
		}
	})
	t.Run("short read on level data breaks loop", func(t *testing.T) {
		buf := new(bytes.Buffer)
		_ = binary.Write(buf, binary.LittleEndian, int32(5)) // claims 5
		// only provide 2 int16s, then truncate
		_ = binary.Write(buf, binary.LittleEndian, int16(11))
		_ = binary.Write(buf, binary.LittleEndian, int16(22))
		br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
		levels, _ := readUncompressedResponseData(br, 10)
		if len(levels) != 5 {
			t.Errorf("levels len = %d, want 5 (allocated even if truncated)", len(levels))
		}
		// First two values should be valid; rest are zero (loop broke early).
		if levels[0] != 0.11 || levels[1] != 0.22 {
			t.Errorf("levels[:2] = %v, want [0.11 0.22]", levels[:2])
		}
		if levels[2] != 0 || levels[3] != 0 || levels[4] != 0 {
			t.Errorf("levels[2:] should be zero, got %v", levels[2:])
		}
	})
}
