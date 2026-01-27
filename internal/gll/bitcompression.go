package gll

// BitCompression implements the S3.Base.Compression.BitCompression algorithm
// used by EASE/AFMG for compressing spectrum data in GLL files.

// Bit mask lookup tables (equivalent to m__E000 in C#)
var bitMasks = []uint32{
	1 << 0, 1 << 1, 1 << 2, 1 << 3, 1 << 4, 1 << 5, 1 << 6, 1 << 7,
	1 << 8, 1 << 9, 1 << 10, 1 << 11, 1 << 12, 1 << 13, 1 << 14, 1 << 15,
	1 << 16, 1 << 17, 1 << 18, 1 << 19, 1 << 20, 1 << 21, 1 << 22, 1 << 23,
	1 << 24, 1 << 25, 1 << 26, 1 << 27, 1 << 28, 1 << 29, 1 << 30, 1 << 31,
}

// Sign extension masks (equivalent to m__E001 in C#)
var signExtendMasks = []int16{
	0,
	-32768, // -(2^15)
	-16384, // -(2^14)
	-8192,  // -(2^13)
	-4096,  // -(2^12)
	-2048,  // -(2^11)
	-1024,  // -(2^10)
	-512,   // -(2^9)
	-256,   // -(2^8)
	-128,   // -(2^7)
	-64,    // -(2^6)
	-32,    // -(2^5)
	-16,    // -(2^4)
	-8,     // -(2^3)
	-4,     // -(2^2)
	-2,     // -(2^1)
	-1,     // -(2^0)
}

// DecompressByteArray decompresses a byte array that was compressed with BitCompression.
// Parameters:
//   - compressedData: the compressed byte array
//   - nDataSamples: expected number of output samples
//   - differentiated: if true, values are stored as deltas and need integration
//   - fixedStepSize: number of values per compression group (typically 8)
//
// Returns the decompressed int16 array.
func DecompressByteArray(compressedData []byte, nDataSamples int, differentiated bool, fixedStepSize int) []int16 {
	data := readArrayBitCompressed(compressedData, nDataSamples, fixedStepSize)

	if differentiated {
		data = integrate(data)
	}

	return data
}

// readArrayBitCompressed reads bit-compressed data from a byte array.
// This is equivalent to ReadArrayBitCompressed(byte[]...) in C#.
func readArrayBitCompressed(compressedData []byte, nDataSamples int, fixedStepSize int) []int16 {
	data := make([]int16, nDataSamples)
	bitPos := 0

	for i := 0; i < nDataSamples; i += fixedStepSize {
		// Determine how many elements in this group (last group may be smaller)
		groupSize := fixedStepSize
		if i+fixedStepSize > nDataSamples {
			groupSize = nDataSamples - i
		}

		// Read 4-bit header: bit depth minus 1
		header := readBitsFromBytes(compressedData, bitPos, 4)
		bitDepth := int(header) + 1
		bitPos += 4

		// Read each value in the group
		for j := 0; j < groupSize; j++ {
			rawValue := readBitsFromBytes(compressedData, bitPos, bitDepth)
			bitPos += bitDepth

			// Sign-extend the value
			data[i+j] = signExtend(rawValue, bitDepth)
		}
	}

	return data
}

// readBitsFromBytes reads nBits from a byte array starting at bitPos.
func readBitsFromBytes(data []byte, bitPos int, nBits int) int16 {
	var result int16

	for i := range nBits {
		byteOffset := (bitPos + i) / 8
		bitOffset := (bitPos + i) % 8

		if byteOffset < len(data) && (data[byteOffset]&byte(bitMasks[bitOffset])) != 0 {
			//nolint:gosec // G115: Intentional conversion for bit manipulation
			result |= int16(bitMasks[i])
		}
	}

	return result
}

// signExtend performs sign extension for a value with the given bit depth.
func signExtend(value int16, bitDepth int) int16 {
	if bitDepth >= 16 {
		return value
	}

	// Check if sign bit is set
	//nolint:gosec // G115: Intentional conversion for bit manipulation
	signBit := int16(bitMasks[bitDepth-1])
	if (value & signBit) != 0 {
		// Sign bit is set, need to extend with 1s
		return value | signExtendMasks[16-bitDepth]
	}

	// Sign bit is 0, clear upper bits (they should already be 0)
	return value & ^signExtendMasks[16-bitDepth]
}

// integrate performs cumulative sum (integration) of delta-encoded values.
// This is equivalent to _E001 in C#.
func integrate(data []int16) []int16 {
	result := make([]int16, len(data))
	if len(data) == 0 {
		return result
	}

	result[0] = data[0]
	for i := 1; i < len(data); i++ {
		// Handle overflow like C# does
		sum := int(data[i]) + int(result[i-1])
		if sum > 32767 {
			sum -= 65536
		} else if sum < -32768 {
			sum += 65536
		}

		//nolint:gosec // G115: Intentional conversion with overflow protection above
		result[i] = int16(sum)
	}

	return result
}
