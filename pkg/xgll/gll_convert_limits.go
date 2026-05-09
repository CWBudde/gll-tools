package xgll

import (
	"fmt"
	"strings"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// Note: hand-edited XGLL fixtures may use legacy Limit/Warning syntax that
// doesn't match the round-trip form emitted by buildLimitStatements /
// buildWarningStatements. The parsers below skip statements they cannot
// interpret rather than failing the entire document.

// parseLimitStatements reads XGLL Limit statements (and optional
// BinaryLimit blobs) back into a slice of Limit. The blob is the source of
// truth when present; metadata-only fallback is used otherwise.
func parseLimitStatements(statements []Statement) ([]gllbin.Limit, error) {
	countStmt := firstStatement(statements, kwLimits)
	if countStmt == nil {
		return nil, nil
	}

	if int(numberArg(countStmt.Args, 0)) <= 0 {
		return nil, nil
	}

	var (
		limits  []gllbin.Limit
		current *gllbin.Limit
	)

	flush := func() {
		if current != nil {
			limits = append(limits, *current)
			current = nil
		}
	}

	for i := range statements {
		stmt := statements[i]
		switch strings.ToLower(stmt.Keyword) {
		case strings.ToLower(kwLimit):
			flush()
			// Accept the round-trip form: <frame>, <type-num>, <boxType>, <value>.
			// Hand-edited legacy XGLL may have fewer or differently typed
			// args; tolerate by skipping unrecognized shapes.
			if len(stmt.Args) < 4 {
				continue
			}
			//nolint:gosec // value is small enum
			current = &gllbin.Limit{
				Frame:      stringArg(stmt.Args, 0),
				Type:       gllbin.LimitType(int32(numberArg(stmt.Args, 1))),
				BoxType:    stringArg(stmt.Args, 2),
				LimitValue: numberArg(stmt.Args, 3),
			}

		case "binarylimit":
			if current == nil {
				continue
			}
			raw, err := decodeBase64(stringArg(stmt.Args, 0))
			if err != nil {
				return nil, fmt.Errorf("decode binary limit: %w", err)
			}
			parsed, err := gllbin.ParseLimitBytes(raw)
			if err != nil {
				return nil, fmt.Errorf("parse binary limit: %w", err)
			}
			if parsed != nil {
				parsed.RawBlock = raw
				current = parsed
			}
		}
	}

	flush()

	return limits, nil
}

// parseWarningStatements reads XGLL Warning statements (and optional
// BinaryWarning blobs) back into a slice of Warning.
func parseWarningStatements(statements []Statement) ([]gllbin.Warning, error) {
	countStmt := firstStatement(statements, kwWarnings)
	if countStmt == nil {
		return nil, nil
	}

	if int(numberArg(countStmt.Args, 0)) <= 0 {
		return nil, nil
	}

	var (
		warnings []gllbin.Warning
		current  *gllbin.Warning
	)

	flush := func() {
		if current != nil {
			warnings = append(warnings, *current)
			current = nil
		}
	}

	for i := range statements {
		stmt := statements[i]
		switch strings.ToLower(stmt.Keyword) {
		case strings.ToLower(kwWarning):
			flush()
			if len(stmt.Args) < 4 {
				continue
			}
			//nolint:gosec // value is small enum
			current = &gllbin.Warning{
				Frame:      stringArg(stmt.Args, 0),
				Type:       gllbin.WarningType(int32(numberArg(stmt.Args, 1))),
				Text:       unescapeText(stringArg(stmt.Args, 2)),
				LimitValue: numberArg(stmt.Args, 3),
			}

		case "binarywarning":
			if current == nil {
				continue
			}
			raw, err := decodeBase64(stringArg(stmt.Args, 0))
			if err != nil {
				return nil, fmt.Errorf("decode binary warning: %w", err)
			}
			parsed, err := gllbin.ParseWarningBytes(raw)
			if err != nil {
				return nil, fmt.Errorf("parse binary warning: %w", err)
			}
			if parsed != nil {
				parsed.RawBlock = raw
				current = parsed
			}
		}
	}

	flush()

	return warnings, nil
}
