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
	// Validate input
	if file == nil {
		return nil, fmt.Errorf("file is nil")
	}

	// Extract GenSystem basics
	sys := file.GenSystem
	if sys.Label == "" || sys.Key == "" {
		return nil, fmt.Errorf("missing system label/key")
	}

	// Map enum to XGLL system type code
	systemType, err := formatSystemType(sys.Type)
	if err != nil {
		return nil, err
	}

	// Seed required statements
	statements := make([]Statement, 0, 12)
	statements = append(statements, newStatement("GLL"))
	statements = append(statements, newStatement("Format", stringValue("3D")))
	statements = append(statements, newStatement("FormatVersion", numberValue(1.0)))
	// Preserve original binary header details for round-trip testing.
	statements = append(statements, newStatement("BinaryFormatVersion", numberValue(float64(file.Header.FormatVersion))))
	statements = append(statements, newStatement("BinarySubVersion", numberValue(float64(file.Header.SubVersion))))
	// Optional checksum and hash
	if hasChecksum(file.Header.Checksum) {
		statements = append(statements, newStatement("BinaryChecksum", stringValue(hex.EncodeToString(file.Header.Checksum[:]))))
	}
	if hasHash(file.Header.HashID) {
		statements = append(statements, newStatement("BinaryHash", stringValue(hex.EncodeToString(file.Header.HashID[:]))))
	}
	// Optional raw GenSystem block
	if len(sys.RawBlock) > 0 {
		encoded := base64.StdEncoding.EncodeToString(sys.RawBlock)
		statements = append(statements, newStatement("BinaryGenSystem", stringValue(encoded)))
	}
	// Optional GenSystem sub-version
	if sys.SubVersion != 0 {
		statements = append(statements, newStatement("BinaryGenSystemSubVersion", numberValue(float64(sys.SubVersion))))
	}
	// Optional GenSystem flags
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
	// Optional raw database block
	if file.Database != nil && len(file.Database.RawBlock) > 0 {
		encoded := base64.StdEncoding.EncodeToString(file.Database.RawBlock)
		statements = append(statements, newStatement("BinaryDatabase", stringValue(encoded)))
	}
	// Optional tail payload
	if len(file.RawTail) > 0 {
		encoded := base64.StdEncoding.EncodeToString(file.RawTail)
		statements = append(statements, newStatement("BinaryTail", stringValue(encoded)))
	}
	// Primary System line
	statements = append(statements, newStatement("System",
		stringValue(sys.Label),
		stringValue(sys.Key),
		stringValue(systemType),
	))

	// Optional GenSystem fields
	if sys.Version != 0 {
		statements = append(statements, newStatement("SystemVersion", numberValue(sys.Version)))
	}

	if sys.Company != "" {
		statements = append(statements, newStatement("Company", stringValue(sys.Company)))
	}

	// Freeform text fields need escaping
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

	// Optional background color
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
	// Optional database file entries
	if file.Database != nil {
		statements = append(statements, buildDataFileStatements(file.Database)...)
		statements = append(statements, buildBoxTypeStatements(file.Database)...)
		sourceStmts, err := buildSourceDefinitionStatements(file.Database)
		if err != nil {
			return nil, fmt.Errorf("build source definitions: %w", err)
		}
		statements = append(statements, sourceStmts...)
	}

	// Assemble document and validate blocks
	doc := &Document{Statements: statements}
	doc.Blocks = buildBlocks(doc.Statements)
	doc.Diagnostics = append(doc.Diagnostics, validateBlocks(doc)...)

	// Return parsed document
	return doc, nil
}

func formatSystemType(value gllbin.SystemType) (string, error) {
	// Map enum to string code
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
	// Helper to build statements
	return Statement{Keyword: keyword, Args: args}
}

func stringValue(value string) Value {
	// Wrap string in Value
	return Value{Kind: ValueString, Raw: value, Str: value}
}

func numberValue(value float64) Value {
	// Wrap number with formatted text
	return Value{Kind: ValueNumber, Raw: formatNumber(value), Num: value}
}

func escapeText(value string) string {
	// Escape control characters for XGLL
	value = strings.ReplaceAll(value, "\t", "/t")
	value = strings.ReplaceAll(value, "\n", "/n")
	value = strings.ReplaceAll(value, "\r", "/r")

	return value
}

func splitRGB(color int32) (int, int, int) {
	// Unpack RGB from packed int32
	r := int((color >> 16) & 0xFF)
	g := int((color >> 8) & 0xFF)
	b := int(color & 0xFF)

	return r, g, b
}

func formatNumber(value float64) string {
	// Format float without extra zeros
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func buildDataFileStatements(db *gllbin.Database) []Statement {
	// Build XGLL file statements from database
	if db == nil {
		return nil
	}

	var statements []Statement

	// DataFiles header and entries
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

	// IncludeFiles header and entries
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

	// AuthorFiles header and entries
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

	// Return statement list
	return statements
}

func hasChecksum(value [4]byte) bool {
	// Check for non-zero checksum bytes
	for _, b := range value {
		if b != 0 {
			return true
		}
	}

	return false
}

func hasHash(value [32]byte) bool {
	// Check for non-zero hash bytes
	for _, b := range value {
		if b != 0 {
			return true
		}
	}

	return false
}

func buildBoxTypeStatements(db *gllbin.Database) []Statement {
	// Build XGLL BoxType statements from database
	if db == nil || len(db.BoxTypes) == 0 {
		return nil
	}

	var statements []Statement

	// BoxTypes header
	statements = append(statements, newStatement("BoxTypes", numberValue(float64(len(db.BoxTypes)))))

	// Each BoxType
	for _, box := range db.BoxTypes {
		// BoxType declaration
		statements = append(statements, newStatement("BoxType",
			stringValue(box.Label),
			stringValue(box.Key),
		))

		// Sources header and placements
		if len(box.SourcePlacements) > 0 {
			statements = append(statements, newStatement("Sources", numberValue(float64(len(box.SourcePlacements)))))
			for _, src := range box.SourcePlacements {
				statements = append(statements, newStatement("Source",
					stringValue(src.Label),
					stringValue(src.Key),
					numberValue(src.Position.X),
					numberValue(src.Position.Y),
					numberValue(src.Position.Z),
					numberValue(src.Angles.X), // H
					numberValue(src.Angles.Y), // V
					numberValue(src.Angles.Z), // R
					stringValue(src.SourceDefKey),
				))
			}
		}

		// InputConfig
		statements = append(statements, buildInputConfigStatements(box.InputConfig)...)

		// ReferencePoint
		if box.ReferencePoint != nil {
			statements = append(statements, newStatement("ReferencePoint",
				numberValue(box.ReferencePoint.X),
				numberValue(box.ReferencePoint.Y),
				numberValue(box.ReferencePoint.Z),
			))
		}

		// NextPivot
		if box.NextPivot != nil {
			statements = append(statements, newStatement("NextPivot",
				numberValue(box.NextPivot.X),
				numberValue(box.NextPivot.Y),
				numberValue(box.NextPivot.Z),
			))
		}

		// CenterOfMass
		if box.CenterOfMass != nil {
			statements = append(statements, newStatement("CenterOfMass",
				numberValue(box.CenterOfMass.X),
				numberValue(box.CenterOfMass.Y),
				numberValue(box.CenterOfMass.Z),
			))
		}

		// Weight
		if box.Weight != 0 {
			statements = append(statements, newStatement("Weight", numberValue(box.Weight)))
		}

		// Opening angles (if present)
		if box.VerticalOpeningAngle != 0 {
			statements = append(statements, newStatement("VerticalOpeningAngle", numberValue(box.VerticalOpeningAngle)))
		}
		if box.HorizontalOpeningAngle != 0 {
			statements = append(statements, newStatement("HorizontalOpeningAngle", numberValue(box.HorizontalOpeningAngle)))
		}
	}

	return statements
}

// buildInputConfigStatements generates XGLL statements for BoxInputConfig.
func buildInputConfigStatements(cfg *gllbin.BoxInputConfig) []Statement {
	if cfg == nil {
		return nil
	}

	var statements []Statement

	// Input Configurations header (always 1 for embedded config)
	statements = append(statements, newStatement("Input Configurations", numberValue(1)))

	// Input Configuration declaration
	statements = append(statements, newStatement("Input Configuration",
		stringValue(cfg.Label),
		stringValue(cfg.Key),
	))

	// Inputs header and items
	if len(cfg.Inputs) > 0 {
		statements = append(statements, newStatement("Inputs", numberValue(float64(len(cfg.Inputs)))))

		for _, input := range cfg.Inputs {
			// Input declaration
			statements = append(statements, newStatement("Input", stringValue(input.Label)))

			// Rated Impedance (only if non-zero)
			if input.RatedImpedance != 0 {
				statements = append(statements, newStatement("Rated Impedance", numberValue(input.RatedImpedance)))
			}

			// Links header and items
			if len(input.SourceLinks) > 0 {
				statements = append(statements, newStatement("Links", numberValue(float64(len(input.SourceLinks)))))

				for _, link := range input.SourceLinks {
					statements = append(statements, newStatement("Link",
						stringValue(link.SourceKey),
						stringValue(link.FilterGrpKey),
					))
				}
			}
		}
	}

	return statements
}
