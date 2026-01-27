package gll

import (
	"bytes"
	"encoding/binary"
	"testing"

	internalgll "github.com/cwbudde/gll-tools/internal/gll"
)

func TestParseCLogSpectrumLPUncompressed(t *testing.T) {
	body := &bytes.Buffer{}
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	_ = binary.Write(body, binary.LittleEndian, int32(3))
	_ = binary.Write(body, binary.LittleEndian, float64(100))
	_ = binary.Write(body, binary.LittleEndian, int32(2))
	_ = binary.Write(body, binary.LittleEndian, int32(0)) // compression type

	_ = binary.Write(body, binary.LittleEndian, int32(2))
	_ = binary.Write(body, binary.LittleEndian, int16(100))
	_ = binary.Write(body, binary.LittleEndian, int16(-50))

	_ = binary.Write(body, binary.LittleEndian, int32(2))
	_ = binary.Write(body, binary.LittleEndian, int16(1000))
	_ = binary.Write(body, binary.LittleEndian, int16(-2000))

	_ = binary.Write(body, binary.LittleEndian, float64(0.25))

	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int32(4+body.Len()))
	_, _ = buf.Write(body.Bytes())

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	tf, err := parseCLogSpectrumLP(br)
	if err != nil {
		t.Fatalf("parseCLogSpectrumLP error: %v", err)
	}

	if len(tf.Level) != 2 || len(tf.Phase) != 2 {
		t.Fatalf("unexpected lengths: level=%d phase=%d", len(tf.Level), len(tf.Phase))
	}

	if tf.Level[0] != 1.0 || tf.Level[1] != -0.5 {
		t.Fatalf("unexpected level values: %v", tf.Level)
	}

	if tf.Phase[0] != 1.0 || tf.Phase[1] != -2.0 {
		t.Fatalf("unexpected phase values: %v", tf.Phase)
	}

	if tf.Delay != 0.25 {
		t.Fatalf("Delay = %f, want 0.25", tf.Delay)
	}
}

func TestParseTransferFunctionLsPsUncompressed(t *testing.T) {
	buildRecord := func(values []int16) []byte {
		body := &bytes.Buffer{}
		_ = binary.Write(body, binary.LittleEndian, int16(0))
		_ = binary.Write(body, binary.LittleEndian, int16(0))
		_ = binary.Write(body, binary.LittleEndian, int32(0)) // compression type
		_ = binary.Write(body, binary.LittleEndian, int32(len(values)))
		for _, v := range values {
			_ = binary.Write(body, binary.LittleEndian, v)
		}

		buf := &bytes.Buffer{}
		_ = binary.Write(buf, binary.LittleEndian, int32(4+body.Len()))
		_, _ = buf.Write(body.Bytes())

		return buf.Bytes()
	}

	complexBody := &bytes.Buffer{}
	_ = binary.Write(complexBody, binary.LittleEndian, int16(0))
	_ = binary.Write(complexBody, binary.LittleEndian, int16(0))
	_, _ = complexBody.Write(buildRecord([]int16{100, -50}))
	_, _ = complexBody.Write(buildRecord([]int16{1000, -2000}))

	complexBuf := &bytes.Buffer{}
	_ = binary.Write(complexBuf, binary.LittleEndian, int32(4+complexBody.Len()))
	_, _ = complexBuf.Write(complexBody.Bytes())

	body := &bytes.Buffer{}
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	_ = binary.Write(body, binary.LittleEndian, int16(0))
	_ = binary.Write(body, binary.LittleEndian, int32(3))
	_ = binary.Write(body, binary.LittleEndian, float64(100))
	_ = binary.Write(body, binary.LittleEndian, int32(2))
	_, _ = body.Write(complexBuf.Bytes())
	_ = binary.Write(body, binary.LittleEndian, float64(0.5))

	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, int32(4+body.Len()))
	_, _ = buf.Write(body.Bytes())

	br := internalgll.NewByteReader(bytes.NewReader(buf.Bytes()))
	tf, err := parseTransferFunctionLsPs(br)
	if err != nil {
		t.Fatalf("parseTransferFunctionLsPs error: %v", err)
	}

	if len(tf.Level) != 2 || len(tf.Phase) != 2 {
		t.Fatalf("unexpected lengths: level=%d phase=%d", len(tf.Level), len(tf.Phase))
	}

	if tf.Level[0] != 1.0 || tf.Level[1] != -0.5 {
		t.Fatalf("unexpected level values: %v", tf.Level)
	}

	if tf.Phase[0] != 1.0 || tf.Phase[1] != -2.0 {
		t.Fatalf("unexpected phase values: %v", tf.Phase)
	}

	if tf.Delay != 0.5 {
		t.Fatalf("Delay = %f, want 0.5", tf.Delay)
	}
}
