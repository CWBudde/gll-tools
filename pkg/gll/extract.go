package gll

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

// ExtractResource extracts a resource from the file
// For zlib resources, returns the compressed data. Use DecompressResource for decompressed data.
func ExtractResource(r io.ReadSeeker, res Resource) ([]byte, error) {
	if _, err := r.Seek(res.Offset, io.SeekStart); err != nil {
		return nil, err
	}

	data := make([]byte, res.Size)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	return data, nil
}

// DecompressResource extracts and decompresses a zlib resource
func DecompressResource(r io.ReadSeeker, res Resource) ([]byte, error) {
	if res.Type != ResourceTypeZlib {
		return ExtractResource(r, res)
	}

	if _, err := r.Seek(res.Offset, io.SeekStart); err != nil {
		return nil, err
	}

	// Read enough data to decompress (use full remaining file as upper bound)
	fileSize, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	if _, err := r.Seek(res.Offset, io.SeekStart); err != nil {
		return nil, err
	}

	compressedData := make([]byte, fileSize-res.Offset)
	if _, err := io.ReadFull(r, compressedData); err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	reader, err := zlib.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, fmt.Errorf("creating zlib reader: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("decompressing: %w", err)
	}

	return decompressed, nil
}

// ExtractDataFile extracts an embedded DataFile from the GLL
func ExtractDataFile(r io.ReadSeeker, df DataFile) ([]byte, error) {
	if _, err := r.Seek(df.Offset, io.SeekStart); err != nil {
		return nil, err
	}

	data := make([]byte, df.Size)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	return data, nil
}

// ExtractIncludeFile extracts an embedded IncludeFile (Additional Data File) from the GLL
// These typically contain PDF documentation, technical drawings, or spec sheets.
func ExtractIncludeFile(r io.ReadSeeker, inc IncludeFile) ([]byte, error) {
	if _, err := r.Seek(inc.Offset, io.SeekStart); err != nil {
		return nil, err
	}

	data := make([]byte, inc.Size)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	return data, nil
}

// CalculateChecksum calculates the GLL checksum for the given data range
func CalculateChecksum(data []byte, start, end int) [4]byte {
	var (
		checksum   [4]byte
		c0, c1, c2 int
	)

	for i := start; i < end && i < len(data); i++ {
		b := int(data[i])
		c0 = (b ^ (123 + c0)) % 256
		c1 = ((11 * b) ^ (1433 + c1)) % 256
		c2 = ((3 * b) ^ (13 + c2)) % 256
	}

	checksum[0] = byte(c0)
	checksum[1] = byte(c1)
	checksum[2] = byte(c2)
	checksum[3] = byte((c0 + c1 + c2) ^ 0x37%256)

	return checksum
}
