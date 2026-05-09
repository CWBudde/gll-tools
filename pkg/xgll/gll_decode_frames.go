package xgll

import (
	"encoding/base64"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// buildFrameStatements emits XGLL statements for the Frames section.
//
// For each Frame:
//
//	"Frame", <Label>, <Key>
//	"TypeFlown", 1                     (when flown)
//	"Weight", <kg>                     (when non-zero)
//	"NextPivot", <x>, <y>, <z>         (when present)
//	"CenterOfMass", <x>, <y>, <z>      (when present)
//	"PinPoints", <count>               (when pin points present)
//	"PinPoint", <Label>, <x>, <y>, <z> (per pin point)
//	"BinaryFrame", <base64>            (when raw block captured)
//
// The BinaryFrame blob preserves the on-disk Frame block (vcheck=1, with
// embedded CaseGeometry) captured during Parse(). XGLL→GLL inflates the
// blob via gllbin.ParseFrameBytes. Frames lacking a blob fall back to the
// metadata-only fields above.
func buildFrameStatements(db *gllbin.Database) []Statement {
	if db == nil || len(db.Frames) == 0 {
		return nil
	}

	var statements []Statement
	statements = append(statements, newStatement(kwFrames, numberValue(float64(len(db.Frames)))))

	for i := range db.Frames {
		frame := db.Frames[i]
		statements = append(statements, newStatement(kwFrame,
			stringValue(frame.Label),
			stringValue(frame.Key),
		))

		if frame.TypeFlown {
			statements = append(statements, newStatement("TypeFlown", numberValue(1)))
		}

		if frame.Weight != 0 {
			statements = append(statements, newStatement("Weight", numberValue(frame.Weight)))
		}

		if frame.NextPivot != nil {
			statements = append(statements, newStatement("NextPivot",
				numberValue(frame.NextPivot.X),
				numberValue(frame.NextPivot.Y),
				numberValue(frame.NextPivot.Z),
			))
		}

		if frame.CenterOfMass != nil {
			statements = append(statements, newStatement("CenterOfMass",
				numberValue(frame.CenterOfMass.X),
				numberValue(frame.CenterOfMass.Y),
				numberValue(frame.CenterOfMass.Z),
			))
		}

		if len(frame.PinPoints) > 0 {
			statements = append(statements, newStatement("PinPoints", numberValue(float64(len(frame.PinPoints)))))
			for _, p := range frame.PinPoints {
				statements = append(statements, newStatement("PinPoint",
					stringValue(p.Label),
					numberValue(p.Vector.X),
					numberValue(p.Vector.Y),
					numberValue(p.Vector.Z),
				))
			}
		}

		if len(frame.RawBlock) > 0 {
			statements = append(statements, newStatement("BinaryFrame",
				stringValue(base64.StdEncoding.EncodeToString(frame.RawBlock)),
			))
		}
	}

	return statements
}
