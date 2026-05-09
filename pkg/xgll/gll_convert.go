package xgll

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

const (
	// Default binary header versions for minimal output
	defaultBinaryFormatVersion int16 = 4
	defaultBinarySubVersion    int16 = 0
)

// BuildGLLFile maps an XGLL document to a minimal GLL model.
func BuildGLLFile(doc *Document) (*gllbin.File, error) {
	// Guard against missing document
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	// Extract required system statement
	sysStmt := firstStatement(doc.Statements, "System")
	if sysStmt == nil || len(sysStmt.Args) < 3 {
		return nil, fmt.Errorf("missing or invalid System line")
	}

	// Parse base system metadata
	label := stringArg(sysStmt.Args, 0)
	key := stringArg(sysStmt.Args, 1)

	sysType, err := parseSystemTypeValue(stringArg(sysStmt.Args, 2))
	if err != nil {
		return nil, err
	}

	// Initialize minimal GLL file and header
	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = defaultBinaryFormatVersion
	file.Header.SubVersion = defaultBinarySubVersion

	// Apply header settings
	if err := applyHeaderSettings(file, doc); err != nil {
		return nil, err
	}

	// Apply parsed system identifiers
	file.GenSystem.Label = label
	file.GenSystem.Key = key
	file.GenSystem.Type = sysType

	// Apply GenSystem settings
	if err := applyGenSystemSettings(file, doc); err != nil {
		return nil, err
	}

	// Apply database settings
	if err := applyDatabaseSettings(file, doc); err != nil {
		return nil, err
	}

	// Optional raw tail payload
	if v := firstStatement(doc.Statements, "BinaryTail"); v != nil {
		raw, err := decodeBase64(stringArg(v.Args, 0))
		if err != nil {
			return nil, fmt.Errorf("binary tail: %w", err)
		}
		file.RawTail = raw
	}

	// Return assembled minimal file
	return file, nil
}

func applyHeaderSettings(file *gllbin.File, doc *Document) error {
	// Optional header overrides
	if v := firstStatement(doc.Statements, "BinaryFormatVersion"); v != nil {
		file.Header.FormatVersion = toInt16(numberArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "BinarySubVersion"); v != nil {
		file.Header.SubVersion = toInt16(numberArg(v.Args, 0))
	}

	// Optional checksum payload
	if v := firstStatement(doc.Statements, "BinaryChecksum"); v != nil {
		checksum, err := decodeHexBytes(stringArg(v.Args, 0), 4)
		if err != nil {
			return fmt.Errorf("binary checksum: %w", err)
		}
		copy(file.Header.Checksum[:], checksum)
	}

	// Optional header hash payload
	if v := firstStatement(doc.Statements, "BinaryHash"); v != nil {
		hash, err := decodeHexBytes(stringArg(v.Args, 0), 32)
		if err != nil {
			return fmt.Errorf("binary hash: %w", err)
		}
		copy(file.Header.HashID[:], hash)
	}
	return nil
}

func applyGenSystemSettings(file *gllbin.File, doc *Document) error {
	// Optional raw GenSystem block
	if v := firstStatement(doc.Statements, "BinaryGenSystem"); v != nil {
		raw, err := decodeBase64(stringArg(v.Args, 0))
		if err != nil {
			return fmt.Errorf("binary gensystem: %w", err)
		}
		file.GenSystem.RawBlock = raw
	}

	// Optional GenSystem fields
	if v := firstStatement(doc.Statements, "SystemVersion"); v != nil {
		file.GenSystem.Version = numberArg(v.Args, 0)
	}

	if v := firstStatement(doc.Statements, "Company"); v != nil {
		file.GenSystem.Company = stringArg(v.Args, 0)
	}

	// Freeform text fields
	if v := firstStatement(doc.Statements, "InfoText"); v != nil {
		file.GenSystem.InfoText = unescapeText(stringArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "CopyrightText"); v != nil {
		file.GenSystem.CopyrightText = unescapeText(stringArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "SupportText"); v != nil {
		file.GenSystem.SupportText = unescapeText(stringArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "WebsiteText"); v != nil {
		file.GenSystem.WebsiteText = stringArg(v.Args, 0)
	}

	if v := firstStatement(doc.Statements, "EmailText"); v != nil {
		file.GenSystem.EmailText = stringArg(v.Args, 0)
	}

	// Optional background color (RGB)
	if v := firstStatement(doc.Statements, "BackgroundColor"); v != nil {
		file.GenSystem.BackgroundColor = parseBackgroundColor(v.Args)
	}

	// Optional GenSystem raw versioning
	if v := firstStatement(doc.Statements, "BinaryGenSystemSubVersion"); v != nil {
		file.GenSystem.SubVersion = toInt16(numberArg(v.Args, 0))
	}

	// Optional GenSystem flags
	if v := firstStatement(doc.Statements, "BinaryGenSystemFlags"); v != nil {
		flags := int32(numberArg(v.Args, 0))
		file.GenSystem.AllowUserDefinedClusterSetup = (flags & 0x01) != 0
		file.GenSystem.EnableForSubArrays = (flags & 0x02) != 0
		file.GenSystem.FlagsPresent = true
	}
	return nil
}

func applyDatabaseSettings(file *gllbin.File, doc *Document) error {
	// Optional raw database block
	if v := firstStatement(doc.Statements, "BinaryDatabase"); v != nil {
		raw, err := decodeBase64(stringArg(v.Args, 0))
		if err != nil {
			return fmt.Errorf("binary database: %w", err)
		}
		if file.Database == nil {
			file.Database = &gllbin.Database{}
		}
		file.Database.RawBlock = raw
	}

	// Extract file list statements
	dataFiles, includeFiles, authorFiles := parseFileStatements(doc.Statements)
	if len(dataFiles) > 0 || len(includeFiles) > 0 || len(authorFiles) > 0 {
		if file.Database == nil {
			file.Database = &gllbin.Database{}
		}
		file.Database.DataFiles = dataFiles
		file.Database.IncludeFiles = includeFiles
		file.Database.AuthorFiles = authorFiles
	}

	// Extract BoxType definitions
	boxTypes, err := parseBoxTypeStatements(doc.Statements)
	if err != nil {
		return fmt.Errorf("parse box types: %w", err)
	}
	if len(boxTypes) > 0 {
		if file.Database == nil {
			file.Database = &gllbin.Database{}
		}
		file.Database.BoxTypes = boxTypes
	}

	// Extract SourceDefinitions
	sources, err := parseSourceDefinitionStatements(doc.Statements)
	if err != nil {
		return fmt.Errorf("parse source definitions: %w", err)
	}
	if len(sources) > 0 {
		if file.Database == nil {
			file.Database = &gllbin.Database{}
		}
		file.Database.SourceDefinitions = sources
	}

	// Extract FilterGroups
	filterGroups, err := parseFilterGroupStatements(doc.Statements)
	if err != nil {
		return fmt.Errorf("parse filter groups: %w", err)
	}
	if len(filterGroups) > 0 {
		if file.Database == nil {
			file.Database = &gllbin.Database{}
		}
		file.Database.FilterGroups = filterGroups
	}

	return nil
}

func firstStatement(statements []Statement, keyword string) *Statement {
	// Find first matching statement by keyword
	for i := range statements {
		if strings.EqualFold(statements[i].Keyword, keyword) {
			return &statements[i]
		}
	}

	return nil
}

func stringArg(args []Value, idx int) string {
	// Extract string-like argument safely
	if idx < 0 || idx >= len(args) {
		return ""
	}

	if args[idx].Kind == ValueString {
		return args[idx].Str
	}

	return args[idx].Raw
}

func numberArg(args []Value, idx int) float64 {
	// Extract numeric argument safely
	if idx < 0 || idx >= len(args) {
		return 0
	}

	if args[idx].Kind == ValueNumber {
		return args[idx].Num
	}

	return 0
}

func parseSystemTypeValue(value string) (gllbin.SystemType, error) {
	// Map text code to system type enum
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "LA":
		return gllbin.SystemTypeLineArray, nil
	case "CL":
		return gllbin.SystemTypeCluster, nil
	case "LS":
		return gllbin.SystemTypeLoudspeaker, nil
	default:
		return 0, fmt.Errorf("unknown system type %q", value)
	}
}

func unescapeText(value string) string {
	// Replace escaped control sequences
	value = strings.ReplaceAll(value, "/t", "\t")
	value = strings.ReplaceAll(value, "/n", "\n")
	value = strings.ReplaceAll(value, "/r", "\r")

	return value
}

func parseBackgroundColor(args []Value) int32 {
	// Parse RGB into packed int32
	if len(args) < 3 {
		return 0
	}

	r := clampColor(numberArg(args, 0))
	g := clampColor(numberArg(args, 1))
	b := clampColor(numberArg(args, 2))

	// nolint:gosec // values are clamped to 0..255
	return int32(r<<16 | g<<8 | b)
}

func clampColor(value float64) int {
	// Clamp to 0..255 and round
	if value <= 1.0 {
		value *= 255.0
	}

	if value < 0 {
		value = 0
	}

	if value > 255 {
		value = 255
	}

	return int(value + 0.5)
}

func toInt16(value float64) int16 {
	// Clamp float to int16 bounds
	if value > float64(int16(^uint16(0)>>1)) {
		return int16(^uint16(0) >> 1)
	}
	if value < float64(-int16(^uint16(0)>>1)-1) {
		return -int16(^uint16(0)>>1) - 1
	}

	return int16(value)
}

func decodeHexBytes(value string, size int) ([]byte, error) {
	// Decode hex string into fixed-size buffer
	value = strings.TrimSpace(value)
	if value == "" {
		return make([]byte, size), nil
	}

	raw, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}

	if len(raw) != size {
		return nil, fmt.Errorf("expected %d bytes, got %d", size, len(raw))
	}

	return raw, nil
}

func decodeBase64(value string) ([]byte, error) {
	// Decode base64 string to raw bytes
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	return base64.StdEncoding.DecodeString(value)
}

func parseFileStatements(statements []Statement) ([]gllbin.DataFile, []gllbin.IncludeFile, []gllbin.DataFile) {
	// Collect data/include/author file statements
	var dataFiles []gllbin.DataFile
	var includeFiles []gllbin.IncludeFile
	var authorFiles []gllbin.DataFile

	// Walk through all statements
	for i := range statements {
		stmt := statements[i]
		switch strings.ToLower(stmt.Keyword) {
		case "datafile":
			// datafile <name> <size>
			if len(stmt.Args) == 0 {
				continue
			}
			filename := stringArg(stmt.Args, 0)
			if filename == "" {
				continue
			}
			size := int32(numberArg(stmt.Args, 1))
			dataFiles = append(dataFiles, gllbin.DataFile{
				Key:      filename,
				Filename: filename,
				Size:     size,
			})
		case "includefile":
			// includefile <label> <key> <filename> <size>
			if len(stmt.Args) < 3 {
				continue
			}
			filename := stringArg(stmt.Args, 2)
			if filename == "" {
				continue
			}
			size := int32(numberArg(stmt.Args, 3))
			includeFiles = append(includeFiles, gllbin.IncludeFile{
				Label:    stringArg(stmt.Args, 0),
				Key:      stringArg(stmt.Args, 1),
				Filename: filename,
				Size:     size,
			})
		case "authorfile":
			// authorfile <filename> <size>
			if len(stmt.Args) == 0 {
				continue
			}
			filename := stringArg(stmt.Args, 0)
			if filename == "" {
				continue
			}
			size := int32(numberArg(stmt.Args, 1))
			authorFiles = append(authorFiles, gllbin.DataFile{
				Key:      filename,
				Filename: filename,
				Size:     size,
			})
		}
	}

	// Return parsed lists
	return dataFiles, includeFiles, authorFiles
}

func parseBoxTypeStatements(statements []Statement) ([]gllbin.BoxType, error) {
	// Find BoxTypes count declaration
	countStmt := firstStatement(statements, "BoxTypes")
	if countStmt == nil {
		return nil, nil
	}

	expectedCount := int(numberArg(countStmt.Args, 0))
	if expectedCount <= 0 {
		return nil, nil
	}

	// Collect all BoxType statements
	var boxTypes []gllbin.BoxType
	for idx, stmt := range statements {
		if !strings.EqualFold(stmt.Keyword, "BoxType") {
			continue
		}

		if len(stmt.Args) < 2 {
			return nil, fmt.Errorf("BoxType statement requires at least 2 args (label, key)")
		}

		box := gllbin.BoxType{
			Label: stringArg(stmt.Args, 0),
			Key:   stringArg(stmt.Args, 1),
		}

		// Parse subsequent statements that belong to this BoxType
		// until we hit another BoxType or a different major keyword
		parseBoxTypeContent(&box, statements, idx+1)

		boxTypes = append(boxTypes, box)
	}

	return boxTypes, nil
}

func parseBoxTypeContent(box *gllbin.BoxType, statements []Statement, startIdx int) {
	// Parse statements that belong to this BoxType
	// Stop at next BoxType, Frame, Connector, or other major keyword
	majorKeywords := map[string]bool{
		kwBoxType: true, kwFrame: true, kwFrames: true,
		kwConnector: true, kwConnectors: true,
		"Limit": true, "Limits": true,
		"Warning": true, "Warnings": true,
		"SourceDefinition": true, "SourceDefinitions": true,
		"FilterGroup": true, "FilterGroups": true,
	}

	for i := startIdx; i < len(statements); i++ {
		stmt := statements[i]
		keyword := strings.ToLower(stmt.Keyword)

		// Stop at major keywords
		if majorKeywords[stmt.Keyword] {
			break
		}

		switch keyword {
		case "sources":
			// Parse source placements
			count := int(numberArg(stmt.Args, 0))
			box.SourcePlacements = make([]gllbin.BoxSource, 0, count)
			box.Sources = make([]string, 0, count)

		case "source":
			// Source placement: "Source", label, key, x, y, z, h, v, r, sourceDefKey
			if len(stmt.Args) < 9 {
				continue
			}
			src := gllbin.BoxSource{
				Label: stringArg(stmt.Args, 0),
				Key:   stringArg(stmt.Args, 1),
				Position: gllbin.Vector3D{
					X: numberArg(stmt.Args, 2),
					Y: numberArg(stmt.Args, 3),
					Z: numberArg(stmt.Args, 4),
				},
				Angles: gllbin.Vector3D{
					X: numberArg(stmt.Args, 5), // H
					Y: numberArg(stmt.Args, 6), // V
					Z: numberArg(stmt.Args, 7), // R
				},
				SourceDefKey: stringArg(stmt.Args, 8),
			}
			box.SourcePlacements = append(box.SourcePlacements, src)
			if src.SourceDefKey != "" {
				box.Sources = append(box.Sources, src.SourceDefKey)
			}

		case "input configurations", "input configuration", "inputs", "input",
			"rated impedance", "links", "link":
			// Delegate to input config parser
			parseInputConfigStatement(box, keyword, stmt.Args)

		case "geometryfiles", "geometryfile":
			// Skip geometry file references (handled separately in real GLL files)
			continue

		case "referencepoint":
			if len(stmt.Args) >= 3 {
				box.ReferencePoint = &gllbin.Vector3D{
					X: numberArg(stmt.Args, 0),
					Y: numberArg(stmt.Args, 1),
					Z: numberArg(stmt.Args, 2),
				}
			}

		case "nextpivot":
			if len(stmt.Args) >= 3 {
				box.NextPivot = &gllbin.Vector3D{
					X: numberArg(stmt.Args, 0),
					Y: numberArg(stmt.Args, 1),
					Z: numberArg(stmt.Args, 2),
				}
			}

		case "centerofmass":
			if len(stmt.Args) >= 3 {
				box.CenterOfMass = &gllbin.Vector3D{
					X: numberArg(stmt.Args, 0),
					Y: numberArg(stmt.Args, 1),
					Z: numberArg(stmt.Args, 2),
				}
			}

		case "weight":
			if len(stmt.Args) >= 1 {
				box.Weight = numberArg(stmt.Args, 0)
			}

		case "verticalopeningangle":
			if len(stmt.Args) >= 1 {
				box.VerticalOpeningAngle = numberArg(stmt.Args, 0)
			}

		case "horizontalopeningangle":
			if len(stmt.Args) >= 1 {
				box.HorizontalOpeningAngle = numberArg(stmt.Args, 0)
			}
		}
	}
}

// parseInputConfigStatement handles a single InputConfig-related statement.
func parseInputConfigStatement(box *gllbin.BoxType, keyword string, args []Value) {
	switch keyword {
	case "input configurations":
		// Count is informational only
		return

	case "input configuration":
		// "Input Configuration", label, key
		if len(args) >= 2 {
			box.InputConfig = &gllbin.BoxInputConfig{
				Label: stringArg(args, 0),
				Key:   stringArg(args, 1),
			}
		}

	case "inputs":
		// "Inputs", count - count is informational, inputs follow
		if box.InputConfig != nil {
			count := int(numberArg(args, 0))
			box.InputConfig.Inputs = make([]gllbin.BoxInput, 0, count)
		}

	case "input":
		// "Input", label - starts a new input
		if box.InputConfig != nil {
			input := gllbin.BoxInput{
				Label: stringArg(args, 0),
			}
			box.InputConfig.Inputs = append(box.InputConfig.Inputs, input)
		}

	case "rated impedance":
		// "Rated Impedance", value - applies to last input
		if box.InputConfig != nil && len(box.InputConfig.Inputs) > 0 {
			idx := len(box.InputConfig.Inputs) - 1
			box.InputConfig.Inputs[idx].RatedImpedance = numberArg(args, 0)
		}

	case "links":
		// "Links", count - count is informational, links follow
		if box.InputConfig != nil && len(box.InputConfig.Inputs) > 0 {
			idx := len(box.InputConfig.Inputs) - 1
			count := int(numberArg(args, 0))
			box.InputConfig.Inputs[idx].SourceLinks = make([]gllbin.SourceFilterLink, 0, count)
		}

	case "link":
		// "Link", sourceKey, filterGrpKey - adds to last input
		if box.InputConfig != nil && len(box.InputConfig.Inputs) > 0 && len(args) >= 2 {
			idx := len(box.InputConfig.Inputs) - 1
			link := gllbin.SourceFilterLink{
				SourceKey:    stringArg(args, 0),
				FilterGrpKey: stringArg(args, 1),
			}
			box.InputConfig.Inputs[idx].SourceLinks = append(box.InputConfig.Inputs[idx].SourceLinks, link)
		}
	}
}
