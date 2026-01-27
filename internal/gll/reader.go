// Package gll provides internal utilities for GLL file parsing.
package gll

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ByteReader wraps a reader with utilities for reading GLL binary structures
type ByteReader struct {
	r      io.ReadSeeker
	offset int64
}

// NewByteReader creates a new ByteReader from an io.ReadSeeker.
// It syncs the internal offset with the current file position.
func NewByteReader(r io.ReadSeeker) *ByteReader {
	// Get the current position in the underlying reader
	pos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		pos = 0
	}

	return &ByteReader{r: r, offset: pos}
}

// ReadBytes reads n bytes from the stream
func (br *ByteReader) ReadBytes(n int) ([]byte, error) {
	buf := make([]byte, n)

	_, err := io.ReadFull(br.r, buf)
	if err != nil {
		return nil, err
	}

	br.offset += int64(n)

	return buf, nil
}

// ReadInt16 reads a 16-bit little-endian integer
func (br *ByteReader) ReadInt16() (int16, error) {
	var v int16

	err := binary.Read(br.r, binary.LittleEndian, &v)
	if err != nil {
		return 0, err
	}

	br.offset += 2

	return v, nil
}

// ReadInt32 reads a 32-bit little-endian integer
func (br *ByteReader) ReadInt32() (int32, error) {
	var v int32

	err := binary.Read(br.r, binary.LittleEndian, &v)
	if err != nil {
		return 0, err
	}

	br.offset += 4

	return v, nil
}

// ReadSingle reads a 32-bit little-endian float (float32)
func (br *ByteReader) ReadSingle() (float32, error) {
	var v float32

	err := binary.Read(br.r, binary.LittleEndian, &v)
	if err != nil {
		return 0, err
	}

	br.offset += 4

	return v, nil
}

// ReadDouble reads a 64-bit little-endian float (float64)
func (br *ByteReader) ReadDouble() (float64, error) {
	var v float64

	err := binary.Read(br.r, binary.LittleEndian, &v)
	if err != nil {
		return 0, err
	}

	br.offset += 8

	return v, nil
}

// ReadString reads a length-prefixed string (int16 length + data)
func (br *ByteReader) ReadString() (string, error) {
	length, err := br.ReadInt16()
	if err != nil {
		return "", err
	}

	if length < 0 {
		return "", fmt.Errorf("negative string length: %d", length)
	}

	if length == 0 {
		return "", nil
	}

	buf, err := br.ReadBytes(int(length))
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

// ReadByte reads a single byte
func (br *ByteReader) ReadByte() (byte, error) {
	buf := make([]byte, 1)

	_, err := io.ReadFull(br.r, buf)
	if err != nil {
		return 0, err
	}

	br.offset++

	return buf[0], nil
}

// Seek sets the offset for the next read operation
func (br *ByteReader) Seek(offset int64, whence int) (int64, error) {
	pos, err := br.r.Seek(offset, whence)
	if err != nil {
		return 0, err
	}

	br.offset = pos

	return pos, nil
}

// Offset returns the current read offset
func (br *ByteReader) Offset() int64 {
	return br.offset
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
