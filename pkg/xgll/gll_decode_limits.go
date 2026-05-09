package xgll

import (
	"encoding/base64"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// buildLimitStatements emits XGLL statements for the Limits section of the
// database.
//
// For each Limit the writer emits:
//
//	"Limit", <Frame>, <Type>, <BoxType>, <LimitValue>
//	"BinaryLimit", <base64>           (when raw block captured)
//
// The BinaryLimit blob is the original on-disk Limit block bytes captured
// during Parse(), letting the XGLL→GLL round-trip recover the limit
// verbatim. Metadata on the Limit line acts as a fallback for hand-edited
// XGLL text.
func buildLimitStatements(db *gllbin.Database) []Statement {
	if db == nil || len(db.Limits) == 0 {
		return nil
	}

	var statements []Statement
	statements = append(statements, newStatement(kwLimits, numberValue(float64(len(db.Limits)))))

	for i := range db.Limits {
		limit := db.Limits[i]
		statements = append(statements, newStatement(kwLimit,
			stringValue(limit.Frame),
			numberValue(float64(int32(limit.Type))),
			stringValue(limit.BoxType),
			numberValue(limit.LimitValue),
		))

		if len(limit.RawBlock) > 0 {
			statements = append(statements, newStatement("BinaryLimit",
				stringValue(base64.StdEncoding.EncodeToString(limit.RawBlock)),
			))
		}
	}

	return statements
}

// buildWarningStatements emits XGLL statements for the Warnings section.
//
//	"Warning", <Frame>, <Type>, <Text>, <LimitValue>
//	"BinaryWarning", <base64>         (when raw block captured)
func buildWarningStatements(db *gllbin.Database) []Statement {
	if db == nil || len(db.Warnings) == 0 {
		return nil
	}

	var statements []Statement
	statements = append(statements, newStatement(kwWarnings, numberValue(float64(len(db.Warnings)))))

	for i := range db.Warnings {
		warning := db.Warnings[i]
		statements = append(statements, newStatement(kwWarning,
			stringValue(warning.Frame),
			numberValue(float64(int32(warning.Type))),
			stringValue(escapeText(warning.Text)),
			numberValue(warning.LimitValue),
		))

		if len(warning.RawBlock) > 0 {
			statements = append(statements, newStatement("BinaryWarning",
				stringValue(base64.StdEncoding.EncodeToString(warning.RawBlock)),
			))
		}
	}

	return statements
}
