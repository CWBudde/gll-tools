package xgll

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// Scale factors matching the parser in pkg/gll/spectrum_parse.go.
const (
	encodeLevelScale = 0.01  // int16 * 0.01 = dB
	encodePhaseScale = 0.001 // int16 * 0.001 = radians
)

// encodeSourceDefinitionsBuffer writes the SourceDefinitions section.
// Format: block(vcheck=0, sver=0) { count(int32) + N × encodeSourceDefinitionItem }
func (e *gllEncoder) encodeSourceDefinitionsBuffer(items []gllbin.SourceDefinitionItem) ([]byte, error) {
	var content bytes.Buffer

	//nolint:gosec // Item count is controlled by database structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(items))); err != nil {
		return nil, fmt.Errorf("write count: %w", err)
	}

	for i, item := range items {
		b, err := e.encodeSourceDefinitionItem(&item)
		if err != nil {
			return nil, fmt.Errorf("encode source %d (%s): %w", i, item.Key, err)
		}
		if _, err := content.Write(b); err != nil {
			return nil, fmt.Errorf("write source %d: %w", i, err)
		}
	}

	return encodeBlock(0, content.Bytes()), nil
}

// encodeSourceDefinitionItem wraps key + SourceDefinition in a block.
// Format: block(vcheck=0, sver=0) { key(string) + SourceDefinition block }
func (e *gllEncoder) encodeSourceDefinitionItem(item *gllbin.SourceDefinitionItem) ([]byte, error) {
	var content bytes.Buffer

	if err := writeString(&content, item.Key); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}

	if item.Definition != nil {
		defBlock, err := e.encodeSourceDefinition(item.Definition)
		if err != nil {
			return nil, fmt.Errorf("encode definition: %w", err)
		}
		if _, err := content.Write(defBlock); err != nil {
			return nil, fmt.Errorf("write definition: %w", err)
		}
	}

	return encodeBlock(0, content.Bytes()), nil
}

// encodeSourceDefinition writes the SourceDefinition block using vcheck=0,
// which selects CLogSpectrumLP format for the on-axis spectrum on read-back.
func (e *gllEncoder) encodeSourceDefinition(def *gllbin.SourceDefinition) ([]byte, error) {
	var content bytes.Buffer

	if err := writeString(&content, def.Label); err != nil {
		return nil, fmt.Errorf("write label: %w", err)
	}
	if err := writeString(&content, def.CompanyLabel); err != nil {
		return nil, fmt.Errorf("write company label: %w", err)
	}
	if err := writeString(&content, def.Description); err != nil {
		return nil, fmt.Errorf("write description: %w", err)
	}
	if err := binary.Write(&content, binary.LittleEndian, def.NominalBandwidthFrom); err != nil {
		return nil, fmt.Errorf("write bandwidth from: %w", err)
	}
	if err := binary.Write(&content, binary.LittleEndian, def.NominalBandwidthTo); err != nil {
		return nil, fmt.Errorf("write bandwidth to: %w", err)
	}
	if err := binary.Write(&content, binary.LittleEndian, int32(def.DataType)); err != nil {
		return nil, fmt.Errorf("write data type: %w", err)
	}

	// BalloonData (or empty block if nil)
	if def.BalloonData != nil {
		balloonBlock, err := e.encodeBalloonData(def.BalloonData)
		if err != nil {
			return nil, fmt.Errorf("encode balloon: %w", err)
		}
		if _, err := content.Write(balloonBlock); err != nil {
			return nil, fmt.Errorf("write balloon: %w", err)
		}
	} else {
		if err := binary.Write(&content, binary.LittleEndian, int32(0)); err != nil {
			return nil, fmt.Errorf("write empty balloon: %w", err)
		}
	}

	// On-axis section: flags(no CLinResponse ref) + reference level + spectrum
	if err := binary.Write(&content, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write on-axis flags: %w", err)
	}
	if err := binary.Write(&content, binary.LittleEndian, def.OnAxisLevel); err != nil {
		return nil, fmt.Errorf("write on-axis level: %w", err)
	}

	// vcheck=0 in the encodeBlock header tells the reader to use parseCLogSpectrumLP
	if def.OnAxisSpectrum != nil {
		tfBlock, err := encodeCLogSpectrumLP(def.OnAxisSpectrum)
		if err != nil {
			return nil, fmt.Errorf("encode on-axis spectrum: %w", err)
		}
		if _, err := content.Write(tfBlock); err != nil {
			return nil, fmt.Errorf("write on-axis spectrum: %w", err)
		}
	} else {
		if err := binary.Write(&content, binary.LittleEndian, int32(0)); err != nil {
			return nil, fmt.Errorf("write empty on-axis spectrum: %w", err)
		}
	}

	return encodeBlock(0, content.Bytes()), nil
}

// encodeBalloonData writes the BalloonData block with vcheck=0 so that
// each response is read back as CLogSpectrumLP on lazy load.
func (e *gllEncoder) encodeBalloonData(balloon *gllbin.BalloonData) ([]byte, error) {
	var content bytes.Buffer

	if err := binary.Write(&content, binary.LittleEndian, balloon.Flags); err != nil {
		return nil, fmt.Errorf("write flags: %w", err)
	}

	resBlock, err := encodeResolutionDescriptor(&balloon.AngularResolution)
	if err != nil {
		return nil, fmt.Errorf("encode resolution: %w", err)
	}
	if _, err := content.Write(resBlock); err != nil {
		return nil, fmt.Errorf("write resolution: %w", err)
	}

	responses := balloon.Responses
	//nolint:gosec // Response count is controlled by BalloonData structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(responses))); err != nil {
		return nil, fmt.Errorf("write response count: %w", err)
	}

	for i, resp := range responses {
		tfBlock, err := encodeCLogSpectrumLP(&resp)
		if err != nil {
			return nil, fmt.Errorf("encode response %d: %w", i, err)
		}
		if _, err := content.Write(tfBlock); err != nil {
			return nil, fmt.Errorf("write response %d: %w", i, err)
		}
	}

	// vcheck=0: responses use CLogSpectrumLP format on read-back
	return encodeBlock(0, content.Bytes()), nil
}

// encodeResolutionDescriptor writes the fixed 32-byte ResolutionDescriptor block.
// The block content is 24 bytes + 8-byte header = 32 bytes total (expected by parser).
func encodeResolutionDescriptor(res *gllbin.ResolutionDescriptor) ([]byte, error) {
	var content bytes.Buffer

	// Symmetry: internal enum → on-disk code
	if err := binary.Write(&content, binary.LittleEndian, reverseSymmetryCode(res.Symmetry)); err != nil {
		return nil, fmt.Errorf("write symmetry: %w", err)
	}

	frontHalf := int32(0)
	if res.FrontHalfOnly {
		frontHalf = 1
	}
	if err := binary.Write(&content, binary.LittleEndian, frontHalf); err != nil {
		return nil, fmt.Errorf("write front half: %w", err)
	}

	if err := binary.Write(&content, binary.LittleEndian, res.MeridianStep); err != nil {
		return nil, fmt.Errorf("write meridian step: %w", err)
	}
	if err := binary.Write(&content, binary.LittleEndian, res.ParallelStep); err != nil {
		return nil, fmt.Errorf("write parallel step: %w", err)
	}

	// content = 4+4+8+8 = 24 bytes; encodeBlock adds 8 bytes header → total 32 ✓
	return encodeBlock(0, content.Bytes()), nil
}

// reverseSymmetryCode converts internal SymmetryType enum to on-disk codes.
// Internal: 0=None, 1=Vertical, 2=Horizontal, 3=Quarter, 4=Axial
// On-disk:  0=Axial, 1=Quarter, 2=Vertical, 3=Horizontal, 4=None
func reverseSymmetryCode(internal int32) int32 {
	switch gllbin.SymmetryType(internal) {
	case gllbin.SymmetryAxial:
		return 0
	case gllbin.SymmetryQuarter:
		return 1
	case gllbin.SymmetryVertical:
		return 2
	case gllbin.SymmetryHorizontal:
		return 3
	case gllbin.SymmetryNone:
		return 4
	default:
		return 4
	}
}

// encodeCLogSpectrumLP writes a TransferFunction in CLogSpectrumLP format.
// Format: block(vcheck=0, sver=0) {
//
//	LogSpectrumDef(bandsPerOctave + startFreq + pointCount)
//	compressionType(0=uncompressed)
//	levelCount + []int16   (level dB * 100)
//	phaseCount + []int16   (phase rad * 1000)
//	delay(float64)
//
// }
func encodeCLogSpectrumLP(tf *gllbin.TransferFunction) ([]byte, error) {
	var content bytes.Buffer

	// LogSpectrumDefinition
	if err := binary.Write(&content, binary.LittleEndian, tf.Definition.BandsPerOctave); err != nil {
		return nil, fmt.Errorf("write bands per octave: %w", err)
	}
	if err := binary.Write(&content, binary.LittleEndian, tf.Definition.StartFreq); err != nil {
		return nil, fmt.Errorf("write start freq: %w", err)
	}
	if err := binary.Write(&content, binary.LittleEndian, tf.Definition.PointCount); err != nil {
		return nil, fmt.Errorf("write point count: %w", err)
	}

	// Compression type: 0 = uncompressed
	if err := binary.Write(&content, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write compression type: %w", err)
	}

	// Level data (dB → int16 * 0.01)
	//nolint:gosec // Level count is controlled by TransferFunction structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(tf.Level))); err != nil {
		return nil, fmt.Errorf("write level count: %w", err)
	}
	for i, v := range tf.Level {
		val := clampInt16(math.Round(v / encodeLevelScale))
		if err := binary.Write(&content, binary.LittleEndian, val); err != nil {
			return nil, fmt.Errorf("write level[%d]: %w", i, err)
		}
	}

	// Phase data (rad → int16 * 0.001)
	//nolint:gosec // Phase count is controlled by TransferFunction structure
	if err := binary.Write(&content, binary.LittleEndian, int32(len(tf.Phase))); err != nil {
		return nil, fmt.Errorf("write phase count: %w", err)
	}
	for i, v := range tf.Phase {
		val := clampInt16(math.Round(v / encodePhaseScale))
		if err := binary.Write(&content, binary.LittleEndian, val); err != nil {
			return nil, fmt.Errorf("write phase[%d]: %w", i, err)
		}
	}

	// Group delay
	if err := binary.Write(&content, binary.LittleEndian, tf.Delay); err != nil {
		return nil, fmt.Errorf("write delay: %w", err)
	}

	return encodeBlock(0, content.Bytes()), nil
}

// clampInt16 converts a float64 to int16, clamping to valid range.
func clampInt16(v float64) int16 {
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v) //nolint:gosec // value is clamped above
}

// SyntheticSource builds a SourceDefinition with flat acoustic response data,
// suitable for testing. Uses axial symmetry with 10° parallel steps (19 points).
// All responses are flat at the given SPL level in dB.
func SyntheticSource(label, key string, spl float64) gllbin.SourceDefinitionItem {
	def := gllbin.LogSpectrumDefinition{
		BandsPerOctave: 3,    // 1/3 octave
		StartFreq:      50.0, // 50 Hz
		PointCount:     21,   // 50 Hz … 5 kHz
	}

	flatTF := func() gllbin.TransferFunction {
		tf := gllbin.TransferFunction{Definition: def}
		tf.Level = make([]float64, def.PointCount)
		tf.Phase = make([]float64, def.PointCount)
		for i := range tf.Level {
			tf.Level[i] = spl
		}
		return tf
	}

	// Axial symmetry: 1 meridian × 19 parallels (0°…180° at 10° steps)
	res := gllbin.ResolutionDescriptor{
		Symmetry:     int32(gllbin.SymmetryAxial),
		MeridianStep: 360.0, // full circle = 1 column
		ParallelStep: 10.0,  // 19 parallels
	}

	responses := make([]gllbin.TransferFunction, res.ParallelCount())
	for i := range responses {
		responses[i] = flatTF()
	}

	return gllbin.SourceDefinitionItem{
		Key: key,
		Definition: &gllbin.SourceDefinition{
			Label:                label,
			NominalBandwidthFrom: 50.0,
			NominalBandwidthTo:   5000.0,
			DataType:             gllbin.DataTypeThirdOctave,
			OnAxisLevel:          spl,
			OnAxisSpectrum:       func() *gllbin.TransferFunction { tf := flatTF(); return &tf }(),
			BalloonData: &gllbin.BalloonData{
				AngularResolution: res,
				Responses:         responses,
			},
		},
	}
}
