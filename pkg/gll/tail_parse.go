package gll

import (
	"encoding/binary"
	"strings"
)

// TailString represents a length-prefixed ASCII string found in tail bytes.
type TailString struct {
	Offset int
	Length int
	Value  string
}

// TailRecord groups nearby TailString entries.
type TailRecord struct {
	Start   int
	End     int
	Strings []TailString
	Label   string
}

// TailPreset is a heuristic mapping of tail record strings.
type TailPreset struct {
	Kind        string `json:"kind,omitempty"`
	InputLabel  string `json:"input_label,omitempty"`
	InputKey    string `json:"input_key,omitempty"`
	PresetLabel string `json:"preset_label,omitempty"`
	PresetKey   string `json:"preset_key,omitempty"`
}

// TailData contains parsed tail structures.
type TailData struct {
	Strings []TailString `json:"strings,omitempty"`
	Records []TailRecord `json:"records,omitempty"`
	Presets []TailPreset `json:"presets,omitempty"`
}

// ParseTailStrings scans data for uint16-length-prefixed printable ASCII strings.
func ParseTailStrings(data []byte, minLen, maxLen int) []TailString {
	if len(data) < 2 {
		return nil
	}

	var out []TailString
	for i := 0; i+2 <= len(data); i++ {
		l := int(binary.LittleEndian.Uint16(data[i:]))
		if l < minLen || l > maxLen {
			continue
		}
		start := i + 2
		end := start + l
		if end > len(data) {
			continue
		}
		if isPrintableASCII(data[start:end]) {
			out = append(out, TailString{Offset: i, Length: l, Value: string(data[start:end])})
		}
	}

	return out
}

// GroupTailStrings groups TailString entries into records by proximity.
func GroupTailStrings(stringsFound []TailString, gap int) []TailRecord {
	if len(stringsFound) == 0 {
		return nil
	}

	var records []TailRecord
	current := TailRecord{
		Start:   stringsFound[0].Offset,
		Strings: []TailString{stringsFound[0]},
	}
	lastEnd := stringsFound[0].Offset + 2 + stringsFound[0].Length

	for i := 1; i < len(stringsFound); i++ {
		s := stringsFound[i]
		if s.Offset-lastEnd > gap {
			current.End = lastEnd
			current.Label = classifyTailRecord(current.Strings)
			records = append(records, current)
			current = TailRecord{Start: s.Offset, Strings: []TailString{s}}
		} else {
			current.Strings = append(current.Strings, s)
		}
		lastEnd = s.Offset + 2 + s.Length
	}

	current.End = lastEnd
	current.Label = classifyTailRecord(current.Strings)
	records = append(records, current)

	return records
}

// ParseTailPresets attempts to map TailRecords to preset-like structures.
func ParseTailPresets(records []TailRecord) []TailPreset {
	if len(records) == 0 {
		return nil
	}

	var presets []TailPreset
	for _, rec := range records {
		preset := TailPreset{Kind: rec.Label}
		for _, s := range rec.Strings {
			text := s.Value
			lower := strings.ToLower(text)
			switch {
			case strings.HasPrefix(lower, "input -"):
				preset.InputLabel = text
			case strings.HasPrefix(text, "key"):
				preset.InputKey = text
			case strings.HasPrefix(text, "Key"):
				preset.PresetKey = text
			case strings.Contains(lower, "crossover -") || strings.Contains(lower, "band pass"):
				preset.PresetLabel = text
			}
		}

		if preset.InputLabel != "" || preset.PresetLabel != "" || preset.InputKey != "" || preset.PresetKey != "" {
			presets = append(presets, preset)
		}
	}

	return presets
}

func classifyTailRecord(stringsFound []TailString) string {
	var hasInput, hasCrossover, hasBandPass bool
	for _, s := range stringsFound {
		text := strings.ToLower(s.Value)
		switch {
		case strings.Contains(text, "input -"):
			hasInput = true
		case strings.Contains(text, "crossover -"):
			hasCrossover = true
		case strings.Contains(text, "band pass"):
			hasBandPass = true
		}
	}

	switch {
	case hasInput && hasCrossover:
		return "crossover"
	case hasInput && hasBandPass:
		return "band-pass"
	case hasInput:
		return "input"
	default:
		return ""
	}
}

func isPrintableASCII(data []byte) bool {
	for _, b := range data {
		if b < 0x20 || b > 0x7E {
			return false
		}
	}
	return true
}
