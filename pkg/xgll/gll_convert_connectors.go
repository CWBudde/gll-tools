package xgll

import (
	"fmt"
	"strings"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// parseConnectorStatements reads XGLL Connector statements (and optional
// BinaryConnector blobs) back into a slice of Connector. The blob is the
// source of truth for the angle list when present; the Angle metadata is a
// fallback for hand-edited XGLL.
func parseConnectorStatements(statements []Statement) ([]gllbin.Connector, error) {
	countStmt := firstStatement(statements, kwConnectors)
	if countStmt == nil {
		return nil, nil
	}

	if int(numberArg(countStmt.Args, 0)) <= 0 {
		return nil, nil
	}

	var (
		connectors []gllbin.Connector
		current    *gllbin.Connector
		fromBlob   bool
	)

	flush := func() {
		if current != nil {
			connectors = append(connectors, *current)
			current = nil
			fromBlob = false
		}
	}

	for i := range statements {
		stmt := statements[i]
		switch strings.ToLower(stmt.Keyword) {
		case strings.ToLower(kwConnector):
			flush()
			// Accept round-trip form (UpperBox, LowerBox, Frame); hand-edited
			// legacy XGLL fixtures may omit Frame — tolerate that shape too.
			if len(stmt.Args) < 2 {
				continue
			}
			current = &gllbin.Connector{
				UpperBox: stringArg(stmt.Args, 0),
				LowerBox: stringArg(stmt.Args, 1),
				Frame:    stringArg(stmt.Args, 2),
			}

		case "angle":
			if current == nil || fromBlob {
				continue
			}
			if len(stmt.Args) < 2 {
				continue
			}
			current.Angles = append(current.Angles, gllbin.LabeledValueD{
				Label: stringArg(stmt.Args, 0),
				Value: numberArg(stmt.Args, 1),
			})

		case "binaryconnector":
			if current == nil {
				continue
			}
			raw, err := decodeBase64(stringArg(stmt.Args, 0))
			if err != nil {
				return nil, fmt.Errorf("decode binary connector: %w", err)
			}
			parsed, err := gllbin.ParseConnectorBytes(raw)
			if err != nil {
				return nil, fmt.Errorf("parse binary connector: %w", err)
			}
			if parsed != nil {
				parsed.RawBlock = raw
				current = parsed
				fromBlob = true
			}
		}
	}

	flush()

	return connectors, nil
}
