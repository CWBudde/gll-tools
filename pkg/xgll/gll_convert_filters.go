package xgll

import (
	"fmt"
	"strings"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// parseFilterGroupStatements reads XGLL FilterGroup statements back into a
// slice of FilterGroup. The expected layout (produced by
// buildFilterGroupStatements):
//
//	"FilterGroups", <count>
//	"FilterGroup", <label>, <key>
//	"IsOverridable", 1                  (optional)
//	"FilterDefinitions", <count>        (optional)
//	"FilterDefinition", <label>, <key>  (per filter)
//	"BinaryFilterGroup", <base64>       (optional, source of truth for filters)
//
// When a BinaryFilterGroup blob is present it inflates the full FilterGroup
// (Label, Key, IsOverridable, Filters); metadata-only statements act as
// fallback for hand-edited XGLL text.
func parseFilterGroupStatements(statements []Statement) ([]gllbin.FilterGroup, error) {
	countStmt := firstStatement(statements, kwFilterGroups)
	if countStmt == nil {
		return nil, nil
	}

	expectedCount := int(numberArg(countStmt.Args, 0))
	if expectedCount <= 0 {
		return nil, nil
	}

	var groups []gllbin.FilterGroup
	for idx, stmt := range statements {
		if !strings.EqualFold(stmt.Keyword, kwFilterGroup) {
			continue
		}

		if len(stmt.Args) < 2 {
			return nil, fmt.Errorf("FilterGroup statement requires <label>, <key>")
		}

		group := gllbin.FilterGroup{
			Label: stringArg(stmt.Args, 0),
			Key:   stringArg(stmt.Args, 1),
		}

		if err := parseFilterGroupContent(&group, statements, idx+1); err != nil {
			return nil, fmt.Errorf("filter group %q: %w", group.Key, err)
		}

		groups = append(groups, group)
	}

	return groups, nil
}

func parseFilterGroupContent(group *gllbin.FilterGroup, statements []Statement, startIdx int) error {
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
		case "binaryfiltergroup":
			raw, err := decodeBase64(stringArg(stmt.Args, 0))
			if err != nil {
				return fmt.Errorf("decode binary filter group: %w", err)
			}
			parsed, err := gllbin.ParseFilterGroupBytes(raw)
			if err != nil {
				return fmt.Errorf("parse binary filter group: %w", err)
			}
			if parsed != nil {
				if parsed.Label != "" {
					group.Label = parsed.Label
				}
				if parsed.Key != "" {
					group.Key = parsed.Key
				}
				group.IsOverridable = parsed.IsOverridable
				group.Filters = parsed.Filters
				group.RawBlock = raw
			}

		case "isoverridable":
			if numberArg(stmt.Args, 0) != 0 {
				group.IsOverridable = true
			}

		case "filterdefinition":
			// Skip if a BinaryFilterGroup blob has already populated filters
			// (the blob is the source of truth for filter bank data).
			if len(group.RawBlock) > 0 {
				continue
			}
			if len(stmt.Args) < 2 {
				continue
			}
			group.Filters = append(group.Filters, gllbin.FilterDefinition{
				Label: stringArg(stmt.Args, 0),
				Key:   stringArg(stmt.Args, 1),
			})
		}
	}

	return nil
}
