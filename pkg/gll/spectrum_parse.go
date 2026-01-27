package gll

import (
	"fmt"
	"io"
	"math"

	"github.com/MeKo-Christian/gll-tools/internal/gll"
)

// LogSpectrumDefinition defines the frequency grid for spectral data
type LogSpectrumDefinition struct {
	BandsPerOctave int32   `json:"bands_per_octave"` // 24=EQTones, 3=EThirds, 1=EOctaves
	StartFreq      float64 `json:"start_freq"`       // Lowest center frequency (Hz)
	PointCount     int32   `json:"point_count"`      // Number of frequency bands
}

// GetResolutionType returns the resolution type based on bands per octave
func (d LogSpectrumDefinition) GetResolutionType() string {
	switch d.BandsPerOctave {
	case 24:
		return "EQTones"
	case 3:
		return "EThirds"
	case 1:
		return "EOctaves"
	default:
		return "Custom"
	}
}

// GetEndFreq calculates the highest frequency based on the definition
func (d LogSpectrumDefinition) GetEndFreq() float64 {
	if d.PointCount <= 0 || d.BandsPerOctave <= 0 {
		return 0
	}

	return d.StartFreq * math.Pow(2, float64(d.PointCount-1)/float64(d.BandsPerOctave))
}

// GetFrequency returns the center frequency for a given band index
func (d LogSpectrumDefinition) GetFrequency(bandIndex int) float64 {
	return d.StartFreq * math.Pow(2, float64(bandIndex)/float64(d.BandsPerOctave))
}

// FrequencyBand represents a standard 1/3 octave band
type FrequencyBand struct {
	Index     int     `json:"index"`     // Band index (1-21)
	Frequency float64 `json:"frequency"` // Center frequency (Hz)
}

// Standard1_3OctaveBands returns the 21 standard 1/3-octave bands (50Hz - 10kHz)
var Standard1_3OctaveBands = []FrequencyBand{
	{1, 50},
	{2, 63},
	{3, 80},
	{4, 100},
	{5, 125},
	{6, 160},
	{7, 200},
	{8, 250},
	{9, 315},
	{10, 400},
	{11, 500},
	{12, 630},
	{13, 800},
	{14, 1000},
	{15, 1250},
	{16, 1600},
	{17, 2000},
	{18, 2500},
	{19, 3150},
	{20, 4000},
	{21, 5000},
}

// parseLogSpectrumDefinition reads a LogSpectrumDefinition from the stream.
// Format: BandsPerOctave(int32) + LowestFrequency(float64) + NumberOfBands(int32)
func parseLogSpectrumDefinition(br *gll.ByteReader) (*LogSpectrumDefinition, error) {
	bandsPerOctave, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading bands per octave: %w", err)
	}

	lowestFreq, err := br.ReadDouble()
	if err != nil {
		return nil, fmt.Errorf("reading lowest frequency: %w", err)
	}

	numberOfBands, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading number of bands: %w", err)
	}

	return &LogSpectrumDefinition{
		BandsPerOctave: bandsPerOctave,
		StartFreq:      lowestFreq,
		PointCount:     numberOfBands,
	}, nil
}

// parseRecord reads a Record (compressed or uncompressed short array) from the stream.
// Returns the decompressed int16 values.
func parseRecord(br *gll.ByteReader) ([]int16, error) {
	// Read block wrapper
	blockSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading record block size: %w", err)
	}

	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid record block size: %d", blockSize)
	}

	startOffset := br.Offset()
	endOffset := startOffset + int64(blockSize) - 4

	// Read version check (expected: 0)
	vcheck, err := br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading record vcheck: %w", err)
	}

	if vcheck != 0 {
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unsupported record version: %d", vcheck)
	}

	// Read sub-version
	_, err = br.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("reading record sver: %w", err)
	}

	// Read compression type: 0=uncompressed, 1=BitCompression
	compressionType, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading compression type: %w", err)
	}

	// Read element count
	elementCount, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("reading element count: %w", err)
	}

	var data []int16

	switch compressionType {
	case 0:
		// Uncompressed: read elementCount int16 values directly
		data = make([]int16, elementCount)
		for i := int32(0); i < elementCount; i++ {
			data[i], err = br.ReadInt16()
			if err != nil {
				_, _ = br.Seek(endOffset, io.SeekStart)
				return nil, fmt.Errorf("reading data[%d]: %w", i, err)
			}
		}

	case 1:
		// BitCompression: read compressed length, then compressed data
		compressedLen, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("reading compressed length: %w", err)
		}

		// Read compressed bytes (each int16 = 2 bytes)
		compressedBytes := make([]byte, compressedLen*2)
		for i := int32(0); i < compressedLen*2; i++ {
			compressedBytes[i], err = br.ReadByte()
			if err != nil {
				_, _ = br.Seek(endOffset, io.SeekStart)
				return nil, fmt.Errorf("reading compressed byte[%d]: %w", i, err)
			}
		}

		// Decompress using BitCompression with differentiation
		data = gll.DecompressByteArray(compressedBytes, int(elementCount), true, 8)

	default:
		_, _ = br.Seek(endOffset, io.SeekStart)
		return nil, fmt.Errorf("unknown compression type: %d", compressionType)
	}

	// Seek to end of block
	_, _ = br.Seek(endOffset, io.SeekStart)

	return data, nil
}
