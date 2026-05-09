package xgll

import (
	"encoding/base64"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// buildFilterGroupStatements emits XGLL statements for the FilterGroups
// section of the database.
//
// For each FilterGroup the writer emits:
//
//	"FilterGroup", <Label>, <Key>
//	"IsOverridable", 1                  (when true)
//	"FilterDefinitions", <count>        (when filters present)
//	"FilterDefinition", <Label>, <Key>  (per filter)
//	"BinaryFilterGroup", <base64>       (when raw block captured)
//
// The BinaryFilterGroup blob is the original on-disk FilterGroup block
// (size header + payload) captured during Parse(). It preserves the full
// filter bank data (IIR/FIR/LogSpectrum) for round-trip without
// reimplementing the FilterGroup binary encoder. FilterGroups parsed from
// XGLL text without a binary blob keep the metadata only.
func buildFilterGroupStatements(db *gllbin.Database) []Statement {
	if db == nil || len(db.FilterGroups) == 0 {
		return nil
	}

	var statements []Statement
	statements = append(statements, newStatement(kwFilterGroups, numberValue(float64(len(db.FilterGroups)))))

	for i := range db.FilterGroups {
		group := db.FilterGroups[i]

		statements = append(statements, newStatement(kwFilterGroup,
			stringValue(group.Label),
			stringValue(group.Key),
		))

		if group.IsOverridable {
			statements = append(statements, newStatement("IsOverridable", numberValue(1)))
		}

		if len(group.Filters) > 0 {
			statements = append(statements, newStatement("FilterDefinitions", numberValue(float64(len(group.Filters)))))
			for _, f := range group.Filters {
				statements = append(statements, newStatement("FilterDefinition",
					stringValue(f.Label),
					stringValue(f.Key),
				))
			}
		}

		if len(group.RawBlock) > 0 {
			statements = append(statements, newStatement("BinaryFilterGroup",
				stringValue(base64.StdEncoding.EncodeToString(group.RawBlock)),
			))
		}
	}

	return statements
}
