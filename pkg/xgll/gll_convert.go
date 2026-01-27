package xgll

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

const (
	defaultBinaryFormatVersion int16 = 4
	defaultBinarySubVersion    int16 = 0
)

// BuildGLLFile maps an XGLL document to a minimal GLL model.
func BuildGLLFile(doc *Document) (*gllbin.File, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	sysStmt := firstStatement(doc.Statements, "System")
	if sysStmt == nil || len(sysStmt.Args) < 3 {
		return nil, fmt.Errorf("missing or invalid System line")
	}

	label := stringArg(sysStmt.Args, 0)
	key := stringArg(sysStmt.Args, 1)

	sysType, err := parseSystemTypeValue(stringArg(sysStmt.Args, 2))
	if err != nil {
		return nil, err
	}

	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = defaultBinaryFormatVersion
	file.Header.SubVersion = defaultBinarySubVersion

	if v := firstStatement(doc.Statements, "BinaryFormatVersion"); v != nil {
		file.Header.FormatVersion = toInt16(numberArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "BinarySubVersion"); v != nil {
		file.Header.SubVersion = toInt16(numberArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "BinaryChecksum"); v != nil {
		checksum, err := decodeHexBytes(stringArg(v.Args, 0), 4)
		if err != nil {
			return nil, fmt.Errorf("binary checksum: %w", err)
		}
		copy(file.Header.Checksum[:], checksum)
	}

	if v := firstStatement(doc.Statements, "BinaryHash"); v != nil {
		hash, err := decodeHexBytes(stringArg(v.Args, 0), 32)
		if err != nil {
			return nil, fmt.Errorf("binary hash: %w", err)
		}
		copy(file.Header.HashID[:], hash)
	}

	if v := firstStatement(doc.Statements, "BinaryGenSystem"); v != nil {
		raw, err := decodeBase64(stringArg(v.Args, 0))
		if err != nil {
			return nil, fmt.Errorf("binary gensystem: %w", err)
		}
		file.GenSystem.RawBlock = raw
	}

	file.GenSystem.Label = label
	file.GenSystem.Key = key
	file.GenSystem.Type = sysType

	if v := firstStatement(doc.Statements, "SystemVersion"); v != nil {
		file.GenSystem.Version = numberArg(v.Args, 0)
	}

	if v := firstStatement(doc.Statements, "Company"); v != nil {
		file.GenSystem.Company = stringArg(v.Args, 0)
	}

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

	if v := firstStatement(doc.Statements, "BackgroundColor"); v != nil {
		file.GenSystem.BackgroundColor = parseBackgroundColor(v.Args)
	}

	if v := firstStatement(doc.Statements, "BinaryGenSystemSubVersion"); v != nil {
		file.GenSystem.SubVersion = toInt16(numberArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "BinaryGenSystemFlags"); v != nil {
		flags := int32(numberArg(v.Args, 0))
		file.GenSystem.AllowUserDefinedClusterSetup = (flags & 0x01) != 0
		file.GenSystem.EnableForSubArrays = (flags & 0x02) != 0
		file.GenSystem.FlagsPresent = true
	}

	if v := firstStatement(doc.Statements, "BinaryDatabase"); v != nil {
		raw, err := decodeBase64(stringArg(v.Args, 0))
		if err != nil {
			return nil, fmt.Errorf("binary database: %w", err)
		}
		if file.Database == nil {
			file.Database = &gllbin.Database{}
		}
		file.Database.RawBlock = raw
	}

	if v := firstStatement(doc.Statements, "BinaryTail"); v != nil {
		raw, err := decodeBase64(stringArg(v.Args, 0))
		if err != nil {
			return nil, fmt.Errorf("binary tail: %w", err)
		}
		file.RawTail = raw
	}

	dataFiles, includeFiles, authorFiles := parseFileStatements(doc.Statements)
	if len(dataFiles) > 0 || len(includeFiles) > 0 || len(authorFiles) > 0 {
		if file.Database == nil {
			file.Database = &gllbin.Database{}
		}
		file.Database.DataFiles = dataFiles
		file.Database.IncludeFiles = includeFiles
		file.Database.AuthorFiles = authorFiles
	}

	return file, nil
}

func firstStatement(statements []Statement, keyword string) *Statement {
	for i := range statements {
		if strings.EqualFold(statements[i].Keyword, keyword) {
			return &statements[i]
		}
	}

	return nil
}

func stringArg(args []Value, idx int) string {
	if idx < 0 || idx >= len(args) {
		return ""
	}

	if args[idx].Kind == ValueString {
		return args[idx].Str
	}

	return args[idx].Raw
}

func numberArg(args []Value, idx int) float64 {
	if idx < 0 || idx >= len(args) {
		return 0
	}

	if args[idx].Kind == ValueNumber {
		return args[idx].Num
	}

	return 0
}

func parseSystemTypeValue(value string) (gllbin.SystemType, error) {
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
	value = strings.ReplaceAll(value, "/t", "\t")
	value = strings.ReplaceAll(value, "/n", "\n")
	value = strings.ReplaceAll(value, "/r", "\r")

	return value
}

func parseBackgroundColor(args []Value) int32 {
	if len(args) < 3 {
		return 0
	}

	r := clampColor(numberArg(args, 0))
	g := clampColor(numberArg(args, 1))
	b := clampColor(numberArg(args, 2))

	return int32(r<<16 | g<<8 | b)
}

func clampColor(value float64) int {
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
	if value > float64(int16(^uint16(0)>>1)) {
		return int16(^uint16(0) >> 1)
	}
	if value < float64(-int16(^uint16(0)>>1)-1) {
		return -int16(^uint16(0)>>1) - 1
	}

	return int16(value)
}

func decodeHexBytes(value string, size int) ([]byte, error) {
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
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	return base64.StdEncoding.DecodeString(value)
}

func parseFileStatements(statements []Statement) ([]gllbin.DataFile, []gllbin.IncludeFile, []gllbin.DataFile) {
	var dataFiles []gllbin.DataFile
	var includeFiles []gllbin.IncludeFile
	var authorFiles []gllbin.DataFile

	for i := range statements {
		stmt := statements[i]
		switch strings.ToLower(stmt.Keyword) {
		case "datafile":
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

	return dataFiles, includeFiles, authorFiles
}
