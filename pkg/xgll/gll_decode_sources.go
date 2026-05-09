package xgll

import (
	"encoding/base64"
	"fmt"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// buildSourceDefinitionStatements emits XGLL statements for the
// SourceDefinitions section of the database.
//
// For each SourceDefinitionItem the writer emits:
//
//	"SourceDefinition", <Label>, <Key>
//	"Bandwidth", <from>, <to>           (when either is non-zero)
//	"DataType", <int>                   (when non-zero)
//	"OnAxisLevel", <level>              (when non-zero)
//	"Company", <company>                (optional)
//	"Description", <description>        (optional)
//	"BinarySourceDefinition", <base64>  (always — preserves balloon/TF data)
//
// The BinarySourceDefinition blob contains the full encodeSourceDefinitionItem
// payload so XGLL → GLL can reconstruct the BalloonData and OnAxisSpectrum
// without reimplementing acoustic data parsing in text form.
func buildSourceDefinitionStatements(db *gllbin.Database) ([]Statement, error) {
	if db == nil || len(db.SourceDefinitions) == 0 {
		return nil, nil
	}

	enc := &gllEncoder{}

	var statements []Statement
	statements = append(statements, newStatement("SourceDefinitions", numberValue(float64(len(db.SourceDefinitions)))))

	for i := range db.SourceDefinitions {
		item := db.SourceDefinitions[i]

		label := ""
		if item.Definition != nil {
			label = item.Definition.Label
		}
		statements = append(statements, newStatement("SourceDefinition",
			stringValue(label),
			stringValue(item.Key),
		))

		if item.Definition != nil {
			def := item.Definition

			if def.NominalBandwidthFrom != 0 || def.NominalBandwidthTo != 0 {
				statements = append(statements, newStatement("Bandwidth",
					numberValue(def.NominalBandwidthFrom),
					numberValue(def.NominalBandwidthTo),
				))
			}

			if def.DataType != 0 {
				statements = append(statements, newStatement("DataType", numberValue(float64(def.DataType))))
			}

			if def.OnAxisLevel != 0 {
				statements = append(statements, newStatement("OnAxisLevel", numberValue(def.OnAxisLevel)))
			}

			if def.CompanyLabel != "" {
				statements = append(statements, newStatement("Company", stringValue(def.CompanyLabel)))
			}

			if def.Description != "" {
				statements = append(statements, newStatement("Description", stringValue(escapeText(def.Description))))
			}
		}

		raw, err := enc.encodeSourceDefinitionItem(&item)
		if err != nil {
			return nil, fmt.Errorf("encode source definition %q: %w", item.Key, err)
		}

		statements = append(statements, newStatement("BinarySourceDefinition",
			stringValue(base64.StdEncoding.EncodeToString(raw)),
		))
	}

	return statements, nil
}
