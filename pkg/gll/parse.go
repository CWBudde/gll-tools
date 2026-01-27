// Package gll provides parsing and extraction for GLL (Generic Loudspeaker Library) files.
//
// GLL is a binary format used by EASE acoustic simulation software to store
// loudspeaker directivity measurements, frequency responses, and metadata.
//
// Basic usage:
//
//	data, _ := os.Open("speaker.gll")
//	file, err := gll.Parse(data)
//	if err != nil { /* handle error */ }
//
//	// Access metadata
//	fmt.Println(file.GenSystem.Company, file.GenSystem.Label)
//
//	// Extract embedded resources
//	for _, res := range file.Resources {
//	    data, _ := gll.ExtractResource(file, res)
//	    // ... use data
//	}
package gll

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/cwbudde/gll-tools/internal/gll"
)

// Parse reads a GLL file from the provided reader
func Parse(r io.ReadSeeker) (*File, error) {
	file := &File{}
	br := gll.NewByteReader(r)

	// Read and validate header
	if err := parseHeader(br, file); err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}

	// Parse GenSystem
	if err := parseGenSystem(br, file); err != nil {
		return nil, fmt.Errorf("gen_system: %w", err)
	}

	if tail, err := readTailBytes(r, br.Offset()); err == nil && len(tail) > 0 {
		file.RawTail = tail
		stringsFound := ParseTailStrings(tail, 4, 200)
		records := GroupTailStrings(stringsFound, 64)
		presets := ParseTailPresets(records)
		if len(stringsFound) > 0 || len(records) > 0 || len(presets) > 0 {
			file.TailData = &TailData{
				Strings: stringsFound,
				Records: records,
				Presets: presets,
			}
		}
		if len(presets) > 0 {
			file.TailPresets = presets
		}
	}

	// Populate Metadata from GenSystem for compatibility
	file.Metadata = Metadata{
		ProductName:  file.GenSystem.Label,
		DisplayName:  file.GenSystem.Label,
		Manufacturer: file.GenSystem.Company,
		Description:  file.GenSystem.InfoText,
		Copyright:    file.GenSystem.CopyrightText,
		Website:      file.GenSystem.WebsiteText,
		Email:        file.GenSystem.EmailText,
	}

	// Scan for embedded resources
	if err := scanResources(r, file); err != nil {
		return nil, fmt.Errorf("resources: %w", err)
	}

	return file, nil
}

func readTailBytes(r io.ReadSeeker, offset int64) ([]byte, error) {
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	return io.ReadAll(r)
}

func parseHeader(br *gll.ByteReader, file *File) error {
	// Read magic (4 bytes)
	magic, err := br.ReadBytes(4)
	if err != nil {
		return fmt.Errorf("reading magic: %w", err)
	}

	if string(magic) != gll.MagicEGLL {
		return fmt.Errorf("invalid magic: expected %q, got %q", gll.MagicEGLL, string(magic))
	}

	file.Header.Magic = string(magic)

	// Read reserved (4 bytes, should be 0)
	_, err = br.ReadInt32()
	if err != nil {
		return fmt.Errorf("reading reserved: %w", err)
	}

	// Read format string
	formatStr, err := br.ReadString()
	if err != nil {
		return fmt.Errorf("reading format string: %w", err)
	}

	if formatStr != gll.MagicEASEGLL {
		return fmt.Errorf("invalid format ID: expected %q, got %q", gll.MagicEASEGLL, formatStr)
	}

	file.Header.FormatID = formatStr

	// Read format version (int16)
	file.Header.FormatVersion, err = br.ReadInt16()
	if err != nil {
		return fmt.Errorf("reading format version: %w", err)
	}

	if file.Header.FormatVersion < 3 || file.Header.FormatVersion > 6 {
		return fmt.Errorf("unsupported format version: %d (expected 3-6)", file.Header.FormatVersion)
	}

	// Read sub-version (int16)
	file.Header.SubVersion, err = br.ReadInt16()
	if err != nil {
		return fmt.Errorf("reading sub-version: %w", err)
	}

	// Read checksum (4 bytes, version >= 4)
	if file.Header.FormatVersion >= 4 {
		for i := 0; i < 4; i++ {
			file.Header.Checksum[i], err = br.ReadByte()
			if err != nil {
				return fmt.Errorf("reading checksum: %w", err)
			}
		}
	}

	// Read hash ID (32 bytes, version >= 6)
	if file.Header.FormatVersion >= 6 {
		hashLen, err := br.ReadInt32()
		if err != nil {
			return fmt.Errorf("reading hash length: %w", err)
		}

		if hashLen > 0 {
			for i := 0; i < int(hashLen) && i < 32; i++ {
				file.Header.HashID[i], err = br.ReadByte()
				if err != nil {
					return fmt.Errorf("reading hash: %w", err)
				}
			}
		}
	}

	return nil
}

func parseGenSystem(br *gll.ByteReader, file *File) error {
	// Read block size
	blockSize, err := br.ReadInt32()
	if err != nil {
		return fmt.Errorf("reading block size: %w", err)
	}

	if blockSize <= 0 {
		return fmt.Errorf("invalid block size: %d", blockSize)
	}

	startOffset := br.Offset()
	blockStart := startOffset - 4
	rawBlock, _ := readRawBlock(br, blockStart, int(blockSize))

	// Read version check (should be 0)
	versionCheck, err := br.ReadInt16()
	if err != nil {
		return fmt.Errorf("reading version check: %w", err)
	}

	if versionCheck != 0 {
		// Skip to end of block
		_, _ = br.Seek(startOffset+int64(blockSize)-4, io.SeekStart)
		return fmt.Errorf("unsupported block version: %d", versionCheck)
	}

	// Read sub-version
	subVersion, err := br.ReadInt16()
	if err != nil {
		return fmt.Errorf("reading sub-version: %w", err)
	}
	file.GenSystem.SubVersion = subVersion
	if len(rawBlock) > 0 {
		file.GenSystem.RawBlock = rawBlock
	}

	// Read Label
	file.GenSystem.Label, err = br.ReadString()
	if err != nil {
		return fmt.Errorf("reading label: %w", err)
	}

	// Read Version
	file.GenSystem.Version, err = br.ReadDouble()
	if err != nil {
		return fmt.Errorf("reading version: %w", err)
	}

	// Read Key
	file.GenSystem.Key, err = br.ReadString()
	if err != nil {
		return fmt.Errorf("reading key: %w", err)
	}

	// Read Type
	typeInt, err := br.ReadInt32()
	if err != nil {
		return fmt.Errorf("reading type: %w", err)
	}

	file.GenSystem.Type = SystemType(typeInt)

	// Read Company
	file.GenSystem.Company, err = br.ReadString()
	if err != nil {
		return fmt.Errorf("reading company: %w", err)
	}

	// Read InfoText
	file.GenSystem.InfoText, err = br.ReadString()
	if err != nil {
		return fmt.Errorf("reading info text: %w", err)
	}

	// Read CopyrightText
	file.GenSystem.CopyrightText, err = br.ReadString()
	if err != nil {
		return fmt.Errorf("reading copyright text: %w", err)
	}

	// Read SupportText
	file.GenSystem.SupportText, err = br.ReadString()
	if err != nil {
		return fmt.Errorf("reading support text: %w", err)
	}

	// Read WebsiteText
	file.GenSystem.WebsiteText, err = br.ReadString()
	if err != nil {
		return fmt.Errorf("reading website text: %w", err)
	}

	// Read EmailText
	file.GenSystem.EmailText, err = br.ReadString()
	if err != nil {
		return fmt.Errorf("reading email text: %w", err)
	}

	// Read BackgroundColor
	file.GenSystem.BackgroundColor, err = br.ReadInt32()
	if err != nil {
		return fmt.Errorf("reading background color: %w", err)
	}

	// Parse Database block
	if err := parseDatabase(br, file); err != nil {
		// Non-fatal: log but continue
		// Database parsing is complex and may fail on some files
		slog.Warn("failed to parse database block", "err", err)
	}

	// Read flags if sub-version >= 1
	if subVersion >= 1 {
		// Flags are at the end of GenSystem after Database
		// Try to read them if we have room
		if br.Offset() < startOffset+int64(blockSize)-4-4 { // -4 for flags, -4 for blockSize
			flags, err := br.ReadInt32()
			if err == nil {
				file.GenSystem.AllowUserDefinedClusterSetup = (flags & 0x01) != 0
				file.GenSystem.EnableForSubArrays = (flags & 0x02) != 0
				file.GenSystem.FlagsPresent = true
			}
		}
	}

	// Seek to end of block
	_, err = br.Seek(startOffset+int64(blockSize)-4, io.SeekStart)
	if err != nil {
		return fmt.Errorf("seeking to block end: %w", err)
	}

	return nil
}
