package gll

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// scanResources scans the GLL file for embedded PNG and zlib resources
func scanResources(r io.ReadSeeker, file *File) error {
	// Get file size
	size, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	// Reset to beginning
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// Read entire file for scanning
	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}

	// Track PNG ranges to exclude from zlib search
	var pngRanges [][2]int64

	// Scan for PNG signatures
	pngSig := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	pngEnd := []byte("IEND")

	offset := 0
	for {
		idx := bytes.Index(data[offset:], pngSig)
		if idx == -1 {
			break
		}

		pngStart := offset + idx

		// Find IEND
		endIdx := bytes.Index(data[pngStart:], pngEnd)
		if endIdx == -1 {
			break
		}

		pngSize := endIdx + 8 // IEND + CRC32

		// Try to find the name (look backwards for path string)
		name := findResourceName(data, pngStart)

		file.Resources = append(file.Resources, Resource{
			Type:   ResourceTypePNG,
			Name:   name,
			Offset: int64(pngStart),
			Size:   int64(pngSize),
		})

		pngRanges = append(pngRanges, [2]int64{int64(pngStart), int64(pngStart + pngSize)})
		offset = pngStart + pngSize
	}

	// Scan for zlib-compressed blocks (outside PNG images)
	zlibSigs := [][]byte{
		{0x78, 0x9C}, // default compression
		{0x78, 0x5E}, // fast compression
		{0x78, 0xDA}, // best compression
	}

	for _, sig := range zlibSigs {
		offset := 0
		for {
			idx := bytes.Index(data[offset:], sig)
			if idx == -1 {
				break
			}

			pos := offset + idx

			// Skip if inside PNG
			inPNG := false

			for _, r := range pngRanges {
				if int64(pos) >= r[0] && int64(pos) <= r[1] {
					inPNG = true
					break
				}
			}

			if !inPNG {
				// Try to decompress to verify it's valid zlib
				reader, err := zlib.NewReader(bytes.NewReader(data[pos:]))
				if err == nil {
					decompressed, err := io.ReadAll(reader)
					reader.Close()

					if err == nil && len(decompressed) > 20 {
						// Valid zlib block - find its compressed size
						compressedSize := findZlibEnd(data[pos:])

						// Identify content type
						contentType := identifyZlibContent(decompressed)

						file.Resources = append(file.Resources, Resource{
							Type:             ResourceTypeZlib,
							Name:             contentType,
							Offset:           int64(pos),
							Size:             int64(compressedSize),
							DecompressedSize: int64(len(decompressed)),
						})
					}
				}
			}

			offset = pos + 1
		}
	}

	return nil
}

// findResourceName looks backwards from offset to find a path-like string
func findResourceName(data []byte, offset int) string {
	// Look backwards for a path-like string
	searchStart := offset - 100
	if searchStart < 0 {
		searchStart = 0
	}

	chunk := data[searchStart:offset]

	// Look for common path patterns
	patterns := []string{".png", ".PNG", ".jpg", ".JPG"}
	for _, pat := range patterns {
		idx := bytes.LastIndex(chunk, []byte(pat))
		if idx != -1 {
			// Find start of path
			start := idx
			for start > 0 && chunk[start-1] != 0 && chunk[start-1] >= 32 {
				start--
			}

			if start < idx {
				return string(chunk[start : idx+len(pat)])
			}
		}
	}

	return ""
}

// findZlibEnd estimates the compressed size of a zlib block
// by scanning for the next zlib signature or end of meaningful data
func findZlibEnd(data []byte) int {
	// Decompress to find actual end
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return len(data)
	}
	defer reader.Close()

	// Read all to consume the compressed data
	_, _ = io.ReadAll(reader)

	// The reader doesn't expose bytes consumed, so estimate from decompression
	// Look for next structure marker after current block
	// Common markers: block size (4-byte int), version check (0x0000)
	for i := 2; i < len(data) && i < 100000; i++ {
		// Check if this looks like end of zlib stream (Adler-32 checksum followed by new structure)
		if i > 10 {
			// Try to find a valid block header after this point
			if data[i] == 0 && i+1 < len(data) && data[i+1] == 0 {
				// Possible version check (0x0000)
				return i
			}
		}
	}

	return len(data)
}

// identifyZlibContent attempts to identify the type of decompressed content
func identifyZlibContent(data []byte) string {
	if len(data) == 0 {
		return "empty"
	}

	// Check for PDF content markers
	if bytes.Contains(data[:gll.Min(200, len(data))], []byte("/CIDInit")) ||
		bytes.Contains(data[:gll.Min(200, len(data))], []byte("begincmap")) {
		return "pdf-cmap"
	}

	if bytes.Contains(data[:gll.Min(100, len(data))], []byte(" g\r\n")) ||
		bytes.Contains(data[:gll.Min(100, len(data))], []byte(" m\r\n")) {
		return "pdf-graphics"
	}

	// Check for font data (OpenType table signatures)
	if len(data) >= 4 {
		sig := string(data[:4])
		switch sig {
		case "DSIG", "EBDT", "EBLC", "GDEF", "GPOS", "GSUB":
			return "font-data"
		}
		// TrueType/OpenType magic
		if data[0] == 0x00 && data[1] == 0x01 && data[2] == 0x00 && data[3] == 0x00 {
			return "font-ttf"
		}
	}

	// Check if it's primarily double values (acoustic data)
	if len(data)%8 == 0 && len(data) >= 168 { // At least 21 doubles (one frequency band)
		// Try reading as doubles and check for reasonable acoustic values
		isAcoustic := true

		for i := 0; i < gll.Min(10, len(data)/8); i++ {
			var v float64
			if err := binary.Read(bytes.NewReader(data[i*8:]), binary.LittleEndian, &v); err != nil {
				isAcoustic = false
				break
			}
			// Acoustic data typically -200 to +200 dB range
			if v < -500 || v > 500 {
				isAcoustic = false
				break
			}
		}

		if isAcoustic {
			return "acoustic-data"
		}
	}

	// Check if it's text-like
	textCount := 0

	for _, b := range data[:gll.Min(100, len(data))] {
		if (b >= 32 && b < 127) || b == '\n' || b == '\r' || b == '\t' {
			textCount++
		}
	}

	if float64(textCount)/float64(gll.Min(100, len(data))) > 0.8 {
		return "text"
	}

	return "binary"
}
