package gll

import (
	"fmt"
)

// SystemType represents the type of loudspeaker system
type SystemType int32

const (
	SystemTypeLineArray   SystemType = 0
	SystemTypeCluster     SystemType = 1
	SystemTypeLoudspeaker SystemType = 2
)

func (t SystemType) String() string {
	switch t {
	case SystemTypeLineArray:
		return "LineArray"
	case SystemTypeCluster:
		return "Cluster"
	case SystemTypeLoudspeaker:
		return "Loudspeaker"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

// Header represents the GLL file header
type Header struct {
	Magic         string   `json:"magic"`
	FormatID      string   `json:"format_id"`
	FormatVersion int16    `json:"format_version"`
	SubVersion    int16    `json:"sub_version"`
	Checksum      [4]byte  `json:"checksum,omitempty"`
	HashID        [32]byte `json:"hash_id,omitempty"`
}

// GenSystem represents the main GLL system container
type GenSystem struct {
	Label                        string     `json:"label"`
	Version                      float64    `json:"version"`
	Key                          string     `json:"key"`
	Type                         SystemType `json:"type"`
	Company                      string     `json:"company"`
	InfoText                     string     `json:"info_text,omitempty"`
	CopyrightText                string     `json:"copyright_text,omitempty"`
	SupportText                  string     `json:"support_text,omitempty"`
	WebsiteText                  string     `json:"website_text,omitempty"`
	EmailText                    string     `json:"email_text,omitempty"`
	BackgroundColor              int32      `json:"background_color,omitempty"`
	AllowUserDefinedClusterSetup bool       `json:"allow_user_defined_cluster_setup,omitempty"`
	EnableForSubArrays           bool       `json:"enable_for_sub_arrays,omitempty"`
	SubVersion                   int16      `json:"sub_version,omitempty"`
	FlagsPresent                 bool       `json:"flags_present,omitempty"`
	RawBlock                     []byte     `json:"raw_block,omitempty"`
}

// Metadata contains the loudspeaker metadata (derived from GenSystem for compatibility)
type Metadata struct {
	ProductName  string `json:"product_name,omitempty"`
	DisplayName  string `json:"display_name,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Description  string `json:"description,omitempty"`
	Copyright    string `json:"copyright,omitempty"`
	Website      string `json:"website,omitempty"`
	Email        string `json:"email,omitempty"`
}

// ResourceType identifies the type of embedded resource
type ResourceType string

const (
	ResourceTypePNG     ResourceType = "PNG"
	ResourceTypeZlib    ResourceType = "ZLIB"
	ResourceTypeUnknown ResourceType = "UNKNOWN"
)

// Resource represents an embedded resource in the GLL file
type Resource struct {
	Type             ResourceType `json:"type"`
	Name             string       `json:"name,omitempty"`
	Offset           int64        `json:"offset"`
	Size             int64        `json:"size"`
	DecompressedSize int64        `json:"decompressed_size,omitempty"` // For zlib resources
}

// File represents a parsed GLL file
type File struct {
	Header      Header       `json:"header"`
	GenSystem   GenSystem    `json:"gen_system"`
	Metadata    Metadata     `json:"metadata"`
	Database    *Database    `json:"database,omitempty"`
	Resources   []Resource   `json:"resources,omitempty"`
	RawTail     []byte       `json:"-"`
	TailData    *TailData    `json:"tail_data,omitempty"`
	TailPresets []TailPreset `json:"tail_presets,omitempty"`
}

// Vector3D represents a 3D coordinate in millimeters
type Vector3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}
