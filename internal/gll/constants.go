// Package gll provides internal utilities for GLL file parsing.
package gll

// Magic bytes and identifiers for GLL file format
const (
	// MagicEGLL is the 4-byte file header signature
	MagicEGLL = "EGLL"

	// MagicEASEGLL is the format identifier string
	MagicEASEGLL = "EASE_GLL"
)

// Resource signatures for embedded content
const (
	// PNGSignature is the PNG file signature
	PNGSignature = "\x89PNG\r\n\x1a\n"

	// PNGEnd marks the end of a PNG file
	PNGEnd = "IEND"

	// ZlibDefault is the default compression signature (0x78, 0x9C)
	ZlibDefault = "\x78\x9C"

	// ZlibFast is the fast compression signature (0x78, 0x5E)
	ZlibFast = "\x78\x5E"

	// ZlibBest is the best compression signature (0x78, 0xDA)
	ZlibBest = "\x78\xDA"
)

// Buffer and size limits
const (
	// MaxResourceScan is the safety limit for resource scanning (100MB)
	MaxResourceScan = 100 * 1024 * 1024
)
