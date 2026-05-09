package xgll

import (
	"fmt"
	"strings"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

// parseFrameStatements reads XGLL Frame statements (and optional
// BinaryFrame blobs) back into a slice of Frame. The blob is the source of
// truth for CaseGeometry/PinPoints when present; metadata-only fallback is
// used for hand-edited XGLL text.
func parseFrameStatements(statements []Statement) ([]gllbin.Frame, error) {
	countStmt := firstStatement(statements, kwFrames)
	if countStmt == nil {
		return nil, nil
	}

	if int(numberArg(countStmt.Args, 0)) <= 0 {
		return nil, nil
	}

	var (
		frames   []gllbin.Frame
		current  *gllbin.Frame
		fromBlob bool
	)

	flush := func() {
		if current != nil {
			frames = append(frames, *current)
			current = nil
			fromBlob = false
		}
	}

	for i := range statements {
		stmt := statements[i]
		switch strings.ToLower(stmt.Keyword) {
		case strings.ToLower(kwFrame):
			flush()
			if len(stmt.Args) < 2 {
				continue
			}
			current = &gllbin.Frame{
				Label: stringArg(stmt.Args, 0),
				Key:   stringArg(stmt.Args, 1),
			}

		case "typeflown":
			if current == nil || fromBlob {
				continue
			}
			current.TypeFlown = numberArg(stmt.Args, 0) != 0

		case "weight":
			if current == nil || fromBlob {
				continue
			}
			current.Weight = numberArg(stmt.Args, 0)

		case "nextpivot":
			if current == nil || fromBlob {
				continue
			}
			if len(stmt.Args) < 3 {
				continue
			}
			current.NextPivot = &gllbin.Vector3D{
				X: numberArg(stmt.Args, 0),
				Y: numberArg(stmt.Args, 1),
				Z: numberArg(stmt.Args, 2),
			}

		case "centerofmass":
			if current == nil || fromBlob {
				continue
			}
			if len(stmt.Args) < 3 {
				continue
			}
			current.CenterOfMass = &gllbin.Vector3D{
				X: numberArg(stmt.Args, 0),
				Y: numberArg(stmt.Args, 1),
				Z: numberArg(stmt.Args, 2),
			}

		case "pinpoint":
			if current == nil || fromBlob {
				continue
			}
			if len(stmt.Args) < 4 {
				continue
			}
			current.PinPoints = append(current.PinPoints, gllbin.LabeledVector3D{
				Label: stringArg(stmt.Args, 0),
				Vector: gllbin.Vector3D{
					X: numberArg(stmt.Args, 1),
					Y: numberArg(stmt.Args, 2),
					Z: numberArg(stmt.Args, 3),
				},
			})

		case "binaryframe":
			if current == nil {
				continue
			}
			raw, err := decodeBase64(stringArg(stmt.Args, 0))
			if err != nil {
				return nil, fmt.Errorf("decode binary frame: %w", err)
			}
			parsed, err := gllbin.ParseFrameBytes(raw)
			if err != nil {
				return nil, fmt.Errorf("parse binary frame: %w", err)
			}
			if parsed != nil {
				parsed.RawBlock = raw
				current = parsed
				fromBlob = true
			}
		}
	}

	flush()

	return frames, nil
}
