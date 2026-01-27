package gll

import (
	"fmt"
	"io"

	"github.com/MeKo-Christian/gll-tools/internal/gll"
)

// TransferFunction represents a frequency-dependent response (level + phase)
type TransferFunction struct {
	Definition LogSpectrumDefinition `json:"definition"`
	Level      []float64             `json:"level"` // dB values
	Phase      []float64             `json:"phase"` // Radians
	Delay      float64               `json:"delay"` // Group delay in seconds
}

// Scale factors for short data
const (
	levelScaleFactor = 0.01  // int16 * 0.01 = dB
	phaseScaleFactor = 0.001 // int16 * 0.001 = radians
)

// parseCLogSpectrumLP parses a CLogSpectrumLP (legacy format, version 0).
// Format: blockSize + vcheck(0-1) + sver + LogSpectrumDef + compressionType + data + delay
func parseCLogSpectrumLP(br *gll.ByteReader) (*TransferFunction, error) {
	tf := &TransferFunction{}

	// Read block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading spectrum block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid spectrum block size: %d", blockSize)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4

	// Read version check (valid: 0 or 1)
	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading spectrum vcheck: %w", err)
	}

	if vcheck < 0 || vcheck > 1 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported spectrum version: %d", vcheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading spectrum sver: %w", err)
	}

	// Read LogSpectrumDefinition
	def, err := parseLogSpectrumDefinition(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading log spectrum definition: %w", err)
	}

	tf.Definition = *def

	// Read compression type: 0=uncompressed, 1=BitCompression
	compressionType, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading compression type: %w", err)
	}

	switch compressionType {
	case 0:
		// Uncompressed format
		// Read level count and data
		levelCount, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("reading level count: %w", err)
		}

		tf.Level = make([]float64, levelCount)
		for i := int32(0); i < levelCount; i++ {
			val, err := br.ReadInt16()
			if err != nil {
				_, _ = br.Seek(endOffset, io.SeekStart)
				return nil, fmt.Errorf("reading level[%d]: %w", i, err)
			}

			tf.Level[i] = float64(val) * levelScaleFactor
		}

		// Read phase count and data
		phaseCount, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("reading phase count: %w", err)
		}

		tf.Phase = make([]float64, phaseCount)
		for i := int32(0); i < phaseCount; i++ {
			val, err := br.ReadInt16()
			if err != nil {
				_, _ = br.Seek(endOffset, io.SeekStart)
				return nil, fmt.Errorf("reading phase[%d]: %w", i, err)
			}

			tf.Phase[i] = float64(val) * phaseScaleFactor
		}

		// Read delay
		tf.Delay, err = br.ReadDouble()
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return nil, fmt.Errorf("reading delay: %w", err)
		}

	case 1:
		// BitCompression format
		// Read level data
		levelCount, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("reading level count: %w", err)
		}

		compressedLevelLen, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("reading compressed level length: %w", err)
		}

		compressedLevelBytes := make([]byte, compressedLevelLen*2)
		for i := int32(0); i < compressedLevelLen*2; i++ {
			compressedLevelBytes[i], err = br.ReadByte()
			if err != nil {
				_, _ = br.Seek(endOffset, io.SeekStart)
				return nil, fmt.Errorf("reading compressed level byte[%d]: %w", i, err)
			}
		}

		levelData := gll.DecompressByteArray(compressedLevelBytes, int(levelCount), true, 8)

		tf.Level = make([]float64, levelCount)
		for i, v := range levelData {
			tf.Level[i] = float64(v) * levelScaleFactor
		}

		// Read phase data
		phaseCount, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("reading phase count: %w", err)
		}

		compressedPhaseLen, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("reading compressed phase length: %w", err)
		}

		compressedPhaseBytes := make([]byte, compressedPhaseLen*2)
		for i := int32(0); i < compressedPhaseLen*2; i++ {
			compressedPhaseBytes[i], err = br.ReadByte()
			if err != nil {
				_, _ = br.Seek(endOffset, io.SeekStart)
				return nil, fmt.Errorf("reading compressed phase byte[%d]: %w", i, err)
			}
		}

		phaseData := gll.DecompressByteArray(compressedPhaseBytes, int(phaseCount), true, 8)

		tf.Phase = make([]float64, phaseCount)
		for i, v := range phaseData {
			tf.Phase[i] = float64(v) * phaseScaleFactor
		}

		// Read delay
		tf.Delay, err = br.ReadDouble()
		if err != nil {
			_, _ = br.Seek(endOffset, io.SeekStart)
			return nil, fmt.Errorf("reading delay: %w", err)
		}

	default:
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unknown compression type: %d", compressionType)
	}

	// Seek to end of block
	_, _ = br.Seek(endOffset, io.SeekStart)

	return tf, nil
}

// parseComplexSequence parses a ComplexSequence block containing two Records.
// Format: blockSize + vcheck(0) + sver + Record(level) + Record(phase)
func parseComplexSequence(br *gll.ByteReader) ([]int16, []int16, error) {
	// Read block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, nil, fmt.Errorf("reading complex sequence block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, nil, fmt.Errorf("invalid complex sequence block size: %d", blockSize)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4

	// Read version check (expected: 0)
	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, nil, fmt.Errorf("reading complex sequence vcheck: %w", err)
	}

	if vcheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, nil, fmt.Errorf("unsupported complex sequence version: %d", vcheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, nil, fmt.Errorf("reading complex sequence sver: %w", err)
	}

	// Read first record (level/magnitude)
	levelData, err := parseRecord(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, nil, fmt.Errorf("reading first record: %w", err)
	}

	// Read second record (phase)
	phaseData, err := parseRecord(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, nil, fmt.Errorf("reading second record: %w", err)
	}

	// Seek to end of block
	_, _ = br.Seek(endOffset, io.SeekStart)

	return levelData, phaseData, nil
}

// parseTransferFunctionLsPs parses a TransferFunction in the newer LsPs format (version 1).
// This uses ComplexSpectrum format: blockSize + vcheck + sver + LogSpectrumDef + ComplexSequence + delay
func parseTransferFunctionLsPs(br *gll.ByteReader) (*TransferFunction, error) {
	tf := &TransferFunction{}

	// Read block wrapper (ComplexSpectrum)
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading transfer function block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid transfer function block size: %d", blockSize)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4

	// Read version check (expected: 0)
	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading transfer function vcheck: %w", err)
	}

	if vcheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported transfer function version: %d", vcheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading transfer function sver: %w", err)
	}

	// Read LogSpectrumDefinition
	def, err := parseLogSpectrumDefinition(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading log spectrum definition: %w", err)
	}

	tf.Definition = *def

	// Read ComplexSequence (contains two Records)
	levelData, phaseData, err := parseComplexSequence(br)
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading complex sequence: %w", err)
	}

	tf.Level = make([]float64, len(levelData))
	for i, v := range levelData {
		tf.Level[i] = float64(v) * levelScaleFactor
	}

	tf.Phase = make([]float64, len(phaseData))
	for i, v := range phaseData {
		tf.Phase[i] = float64(v) * phaseScaleFactor
	}

	// Read delay
	tf.Delay, err = br.ReadDouble()
	if err != nil {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("reading delay: %w", err)
	}

	// Seek to end of block
	_, _ = br.Seek(endOffset, io.SeekStart)

	return tf, nil
}

func LoadBalloonResponses(r io.ReadSeeker, balloon *BalloonData) error {
	if balloon.ResponseCount <= 0 {
		return nil
	}

	if balloon.ResponsesOffset <= 0 {
		return fmt.Errorf("balloon responses offset not set")
	}

	// Seek to the responses offset
	_, err := r.Seek(balloon.ResponsesOffset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("seeking to responses: %w", err)
	}

	// Create a ByteReader for parsing
	br := gll.NewByteReader(r)

	balloon.Responses = make([]TransferFunction, balloon.ResponseCount)

	for i := int32(0); i < balloon.ResponseCount; i++ {
		var (
			tf  *TransferFunction
			err error
		)

		if balloon.ResponseVersion == 0 {
			// Legacy CLogSpectrumLP format
			tf, err = parseCLogSpectrumLP(br)
		} else {
			// Newer TransferFunctionLsPs format
			tf, err = parseTransferFunctionLsPs(br)
		}

		if err != nil {
			return fmt.Errorf("parsing response %d: %w", i, err)
		}

		balloon.Responses[i] = *tf
	}

	return nil
}
