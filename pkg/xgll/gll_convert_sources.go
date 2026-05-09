package xgll

import (
	"fmt"
	"strings"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// parseSourceDefinitionStatements reads XGLL SourceDefinition statements back
// into a slice of SourceDefinitionItem.
//
// The expected statement layout (produced by buildSourceDefinitionStatements):
//
//	"SourceDefinitions", <count>
//	"SourceDefinition", <label>, <key>
//	... per-source metadata statements ...
//	"BinarySourceDefinition", <base64>
//
// When a BinarySourceDefinition blob is present it is the source of truth for
// BalloonData / OnAxisSpectrum / OnAxisLevel. Metadata-only statements are
// applied on top so manual edits to the text form take precedence.
func parseSourceDefinitionStatements(statements []Statement) ([]gllbin.SourceDefinitionItem, error) {
	countStmt := firstStatement(statements, "SourceDefinitions")
	if countStmt == nil {
		return nil, nil
	}

	expectedCount := int(numberArg(countStmt.Args, 0))
	if expectedCount <= 0 {
		return nil, nil
	}

	var items []gllbin.SourceDefinitionItem
	for idx, stmt := range statements {
		if !strings.EqualFold(stmt.Keyword, "SourceDefinition") {
			continue
		}

		if len(stmt.Args) == 0 {
			return nil, fmt.Errorf("SourceDefinition statement requires at least 1 arg (key)")
		}

		// Two argument forms are supported:
		//   "SourceDefinition", <Label>, <Key>     (round-trip output)
		//   "SourceDefinition", <Key>               (legacy / external-file)
		var label, key string
		if len(stmt.Args) >= 2 {
			label = stringArg(stmt.Args, 0)
			key = stringArg(stmt.Args, 1)
		} else {
			key = stringArg(stmt.Args, 0)
		}

		item := gllbin.SourceDefinitionItem{Key: key}

		// Walk subsequent statements until the next major keyword or the next
		// SourceDefinition declaration.
		if err := parseSourceDefinitionContent(&item, statements, idx+1); err != nil {
			return nil, fmt.Errorf("source definition %q: %w", key, err)
		}

		// Skip legacy entries that reference an external file but carry no
		// inline acoustic data — the binary GLL writer cannot serialize a
		// definition-less item, and these fixtures already roundtrip via
		// `BinaryDatabase` blobs when needed.
		if item.Definition == nil {
			continue
		}

		// Fill in label from the declaration line if the binary blob did not
		// already carry it.
		if item.Definition.Label == "" && label != "" {
			item.Definition.Label = label
		}

		items = append(items, item)
	}

	return items, nil
}

func parseSourceDefinitionContent(item *gllbin.SourceDefinitionItem, statements []Statement, startIdx int) error {
	majorKeywords := map[string]bool{
		kwBoxType: true, kwBoxTypes: true,
		kwFrame: true, kwFrames: true,
		kwConnector: true, kwConnectors: true,
		kwLimit: true, kwLimits: true,
		kwWarning: true, kwWarnings: true,
		kwSourceDefinition: true, kwSourceDefinitions: true,
		kwFilterGroup: true, kwFilterGroups: true,
	}

	for i := startIdx; i < len(statements); i++ {
		stmt := statements[i]
		if majorKeywords[stmt.Keyword] {
			break
		}

		switch strings.ToLower(stmt.Keyword) {
		case "binarysourcedefinition":
			raw, err := decodeBase64(stringArg(stmt.Args, 0))
			if err != nil {
				return fmt.Errorf("decode binary source definition: %w", err)
			}
			parsed, err := gllbin.ParseSourceDefinitionItemBytes(raw)
			if err != nil {
				return fmt.Errorf("parse binary source definition: %w", err)
			}
			if parsed != nil {
				if parsed.Key != "" {
					item.Key = parsed.Key
				}
				item.Definition = parsed.Definition
			}

		case "bandwidth":
			if item.Definition == nil {
				item.Definition = &gllbin.SourceDefinition{}
			}
			item.Definition.NominalBandwidthFrom = numberArg(stmt.Args, 0)
			item.Definition.NominalBandwidthTo = numberArg(stmt.Args, 1)

		case "datatype":
			if item.Definition == nil {
				item.Definition = &gllbin.SourceDefinition{}
			}
			//nolint:gosec // value is a small DataType enum
			item.Definition.DataType = gllbin.DataType(int32(numberArg(stmt.Args, 0)))

		case "onaxislevel":
			if item.Definition == nil {
				item.Definition = &gllbin.SourceDefinition{}
			}
			item.Definition.OnAxisLevel = numberArg(stmt.Args, 0)

		case "company":
			if item.Definition == nil {
				item.Definition = &gllbin.SourceDefinition{}
			}
			item.Definition.CompanyLabel = stringArg(stmt.Args, 0)

		case "description":
			if item.Definition == nil {
				item.Definition = &gllbin.SourceDefinition{}
			}
			item.Definition.Description = unescapeText(stringArg(stmt.Args, 0))

		case "file":
			// External source data file reference — not supported for inline
			// round-trip yet. Leave the definition untouched.
			continue
		}
	}

	return nil
}
