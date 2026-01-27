package xgll

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// BuildXGLLDocument maps a parsed GLL file to a minimal XGLL document.
func BuildXGLLDocument(file *gllbin.File) (*Document, error) {
	if file == nil {
		return nil, fmt.Errorf("file is nil")
	}

	sys := file.GenSystem
	if sys.Label == "" || sys.Key == "" {
		return nil, fmt.Errorf("missing system label/key")
	}

	systemType, err := formatSystemType(sys.Type)
	if err != nil {
		return nil, err
	}

	statements := make([]Statement, 0, 12)
	statements = append(statements, newStatement("GLL"))
	statements = append(statements, newStatement("Format", stringValue("3D")))
	statements = append(statements, newStatement("FormatVersion", numberValue(1.0)))
	// Preserve original binary header details for round-trip testing.
	statements = append(statements, newStatement("BinaryFormatVersion", numberValue(float64(file.Header.FormatVersion))))
	statements = append(statements, newStatement("BinarySubVersion", numberValue(float64(file.Header.SubVersion))))
	if hasChecksum(file.Header.Checksum) {
		statements = append(statements, newStatement("BinaryChecksum", stringValue(hex.EncodeToString(file.Header.Checksum[:]))))
	}
	if hasHash(file.Header.HashID) {
		statements = append(statements, newStatement("BinaryHash", stringValue(hex.EncodeToString(file.Header.HashID[:]))))
	}
	if len(sys.RawBlock) > 0 {
		encoded := base64.StdEncoding.EncodeToString(sys.RawBlock)
		statements = append(statements, newStatement("BinaryGenSystem", stringValue(encoded)))
	}
	if sys.SubVersion != 0 {
		statements = append(statements, newStatement("BinaryGenSystemSubVersion", numberValue(float64(sys.SubVersion))))
	}
	if sys.FlagsPresent {
		flags := int32(0)
		if sys.AllowUserDefinedClusterSetup {
			flags |= 0x01
		}
		if sys.EnableForSubArrays {
			flags |= 0x02
		}
		statements = append(statements, newStatement("BinaryGenSystemFlags", numberValue(float64(flags))))
	}
	if file.Database != nil && len(file.Database.RawBlock) > 0 {
		encoded := base64.StdEncoding.EncodeToString(file.Database.RawBlock)
		statements = append(statements, newStatement("BinaryDatabase", stringValue(encoded)))
	}
	if len(file.RawTail) > 0 {
		encoded := base64.StdEncoding.EncodeToString(file.RawTail)
		statements = append(statements, newStatement("BinaryTail", stringValue(encoded)))
	}
	statements = append(statements, newStatement("System",
		stringValue(sys.Label),
		stringValue(sys.Key),
		stringValue(systemType),
	))

	if sys.Version != 0 {
		statements = append(statements, newStatement("SystemVersion", numberValue(sys.Version)))
	}

	if sys.Company != "" {
		statements = append(statements, newStatement("Company", stringValue(sys.Company)))
	}

	if sys.InfoText != "" {
		statements = append(statements, newStatement("InfoText", stringValue(escapeText(sys.InfoText))))
	}

	if sys.CopyrightText != "" {
		statements = append(statements, newStatement("CopyrightText", stringValue(escapeText(sys.CopyrightText))))
	}

	if sys.SupportText != "" {
		statements = append(statements, newStatement("SupportText", stringValue(escapeText(sys.SupportText))))
	}

	if sys.WebsiteText != "" {
		statements = append(statements, newStatement("WebsiteText", stringValue(sys.WebsiteText)))
	}

	if sys.EmailText != "" {
		statements = append(statements, newStatement("EmailText", stringValue(sys.EmailText)))
	}

	if sys.BackgroundColor != 0 {
		r, g, b := splitRGB(sys.BackgroundColor)
		statements = append(statements, newStatement("BackgroundColor",
			numberValue(float64(r)),
			numberValue(float64(g)),
			numberValue(float64(b)),
		))
	}

	// Minimal empty blocks to satisfy validation.
	statements = append(statements, newStatement("Layout"))
	statements = append(statements, newStatement("Data"))
	if file.Database != nil {
		statements = append(statements, buildDataFileStatements(file.Database)...)
	}

	doc := &Document{Statements: statements}
	doc.Blocks = buildBlocks(doc.Statements)
	doc.Diagnostics = append(doc.Diagnostics, validateBlocks(doc)...)

	return doc, nil
}

func formatSystemType(value gllbin.SystemType) (string, error) {
	switch value {
	case gllbin.SystemTypeLineArray:
		return "LA", nil
	case gllbin.SystemTypeCluster:
		return "CL", nil
	case gllbin.SystemTypeLoudspeaker:
		return "LS", nil
	default:
		return "", fmt.Errorf("unknown system type %d", value)
	}
}

func newStatement(keyword string, args ...Value) Statement {
	return Statement{Keyword: keyword, Args: args}
}

func stringValue(value string) Value {
	return Value{Kind: ValueString, Raw: value, Str: value}
}

func numberValue(value float64) Value {
	return Value{Kind: ValueNumber, Raw: formatNumber(value), Num: value}
}

func escapeText(value string) string {
	value = strings.ReplaceAll(value, "\t", "/t")
	value = strings.ReplaceAll(value, "\n", "/n")
	value = strings.ReplaceAll(value, "\r", "/r")

	return value
}

func splitRGB(color int32) (int, int, int) {
	r := int((color >> 16) & 0xFF)
	g := int((color >> 8) & 0xFF)
	b := int(color & 0xFF)

	return r, g, b
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func buildDataFileStatements(db *gllbin.Database) []Statement {
	if db == nil {
		return nil
	}

	var statements []Statement

	if len(db.DataFiles) > 0 {
		statements = append(statements, newStatement("DataFiles", numberValue(float64(len(db.DataFiles)))))
		for _, df := range db.DataFiles {
			if df.Size > 0 {
				statements = append(statements, newStatement("DataFile",
					stringValue(df.Filename),
					numberValue(float64(df.Size)),
				))
			} else {
				statements = append(statements, newStatement("DataFile",
					stringValue(df.Filename),
				))
			}
		}
	}

	if len(db.IncludeFiles) > 0 {
		statements = append(statements, newStatement("IncludeFiles", numberValue(float64(len(db.IncludeFiles)))))
		for _, inc := range db.IncludeFiles {
			args := []Value{
				stringValue(inc.Label),
				stringValue(inc.Key),
				stringValue(inc.Filename),
			}
			if inc.Size > 0 {
				args = append(args, numberValue(float64(inc.Size)))
			}
			statements = append(statements, newStatement("IncludeFile", args...))
		}
	}

	if len(db.AuthorFiles) > 0 {
		statements = append(statements, newStatement("AuthorFiles", numberValue(float64(len(db.AuthorFiles)))))
		for _, af := range db.AuthorFiles {
			if af.Size > 0 {
				statements = append(statements, newStatement("AuthorFile",
					stringValue(af.Filename),
					numberValue(float64(af.Size)),
				))
			} else {
				statements = append(statements, newStatement("AuthorFile",
					stringValue(af.Filename),
				))
			}
		}
	}

	return statements
}

func hasChecksum(value [4]byte) bool {
	for _, b := range value {
		if b != 0 {
			return true
		}
	}

	return false
}

func hasHash(value [32]byte) bool {
	for _, b := range value {
		if b != 0 {
			return true
		}
	}

	return false
}
