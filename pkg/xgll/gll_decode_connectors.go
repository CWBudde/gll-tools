package xgll

import (
	"encoding/base64"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// buildConnectorStatements emits XGLL statements for the Connectors section.
//
// For each Connector:
//
//	"Connector", <UpperBox>, <LowerBox>, <Frame>
//	"Angles", <count>                  (when angles present)
//	"Angle", <Label>, <Value>          (per angle)
//	"BinaryConnector", <base64>        (when raw block captured)
//
// The BinaryConnector blob preserves the on-disk Connector block (vcheck=1)
// captured during Parse(), so XGLL→GLL recovers the LabeledValueD angle
// list verbatim. Connectors lacking a blob fall back to the Angle metadata.
func buildConnectorStatements(db *gllbin.Database) []Statement {
	if db == nil || len(db.Connectors) == 0 {
		return nil
	}

	var statements []Statement
	statements = append(statements, newStatement(kwConnectors, numberValue(float64(len(db.Connectors)))))

	for i := range db.Connectors {
		connector := db.Connectors[i]
		statements = append(statements, newStatement(kwConnector,
			stringValue(connector.UpperBox),
			stringValue(connector.LowerBox),
			stringValue(connector.Frame),
		))

		if len(connector.Angles) > 0 {
			statements = append(statements, newStatement("Angles", numberValue(float64(len(connector.Angles)))))
			for _, a := range connector.Angles {
				statements = append(statements, newStatement("Angle",
					stringValue(a.Label),
					numberValue(a.Value),
				))
			}
		}

		if len(connector.RawBlock) > 0 {
			statements = append(statements, newStatement("BinaryConnector",
				stringValue(base64.StdEncoding.EncodeToString(connector.RawBlock)),
			))
		}
	}

	return statements
}
