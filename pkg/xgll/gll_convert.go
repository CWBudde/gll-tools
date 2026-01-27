package xgll

import (
	"fmt"
	"strings"

	gllbin "github.com/MeKo-Christian/gll-tools/pkg/gll"
)

const (
	defaultBinaryFormatVersion int16 = 4
	defaultBinarySubVersion    int16 = 0
)

// BuildGLLFile maps an XGLL document to a minimal GLL model.
func BuildGLLFile(doc *Document) (*gllbin.File, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	sysStmt := firstStatement(doc.Statements, "System")
	if sysStmt == nil || len(sysStmt.Args) < 3 {
		return nil, fmt.Errorf("missing or invalid System line")
	}

	label := stringArg(sysStmt.Args, 0)
	key := stringArg(sysStmt.Args, 1)

	sysType, err := parseSystemTypeValue(stringArg(sysStmt.Args, 2))
	if err != nil {
		return nil, err
	}

	file := &gllbin.File{}
	file.Header.Magic = "EGLL"
	file.Header.FormatID = "EASE_GLL"
	file.Header.FormatVersion = defaultBinaryFormatVersion
	file.Header.SubVersion = defaultBinarySubVersion

	file.GenSystem.Label = label
	file.GenSystem.Key = key
	file.GenSystem.Type = sysType

	if v := firstStatement(doc.Statements, "SystemVersion"); v != nil {
		file.GenSystem.Version = numberArg(v.Args, 0)
	}

	if v := firstStatement(doc.Statements, "Company"); v != nil {
		file.GenSystem.Company = stringArg(v.Args, 0)
	}

	if v := firstStatement(doc.Statements, "InfoText"); v != nil {
		file.GenSystem.InfoText = unescapeText(stringArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "CopyrightText"); v != nil {
		file.GenSystem.CopyrightText = unescapeText(stringArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "SupportText"); v != nil {
		file.GenSystem.SupportText = unescapeText(stringArg(v.Args, 0))
	}

	if v := firstStatement(doc.Statements, "WebsiteText"); v != nil {
		file.GenSystem.WebsiteText = stringArg(v.Args, 0)
	}

	if v := firstStatement(doc.Statements, "EmailText"); v != nil {
		file.GenSystem.EmailText = stringArg(v.Args, 0)
	}

	if v := firstStatement(doc.Statements, "BackgroundColor"); v != nil {
		file.GenSystem.BackgroundColor = parseBackgroundColor(v.Args)
	}

	return file, nil
}

func firstStatement(statements []Statement, keyword string) *Statement {
	for i := range statements {
		if strings.EqualFold(statements[i].Keyword, keyword) {
			return &statements[i]
		}
	}

	return nil
}

func stringArg(args []Value, idx int) string {
	if idx < 0 || idx >= len(args) {
		return ""
	}

	if args[idx].Kind == ValueString {
		return args[idx].Str
	}

	return args[idx].Raw
}

func numberArg(args []Value, idx int) float64 {
	if idx < 0 || idx >= len(args) {
		return 0
	}

	if args[idx].Kind == ValueNumber {
		return args[idx].Num
	}

	return 0
}

func parseSystemTypeValue(value string) (gllbin.SystemType, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "LA":
		return gllbin.SystemTypeLineArray, nil
	case "CL":
		return gllbin.SystemTypeCluster, nil
	case "LS":
		return gllbin.SystemTypeLoudspeaker, nil
	default:
		return 0, fmt.Errorf("unknown system type %q", value)
	}
}

func unescapeText(value string) string {
	value = strings.ReplaceAll(value, "/t", "\t")
	value = strings.ReplaceAll(value, "/n", "\n")
	value = strings.ReplaceAll(value, "/r", "\r")

	return value
}

func parseBackgroundColor(args []Value) int32 {
	if len(args) < 3 {
		return 0
	}

	r := clampColor(numberArg(args, 0))
	g := clampColor(numberArg(args, 1))
	b := clampColor(numberArg(args, 2))

	return int32(r<<16 | g<<8 | b)
}

func clampColor(value float64) int {
	if value <= 1.0 {
		value *= 255.0
	}

	if value < 0 {
		value = 0
	}

	if value > 255 {
		value = 255
	}

	return int(value + 0.5)
}
