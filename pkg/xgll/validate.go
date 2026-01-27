package xgll

import (
	"strconv"
	"strings"
)

// ValidateSystemConstraints checks block presence and system type consistency.
func ValidateSystemConstraints(doc *Document) []Diagnostic {
	var diags []Diagnostic
	if doc == nil || len(doc.Statements) == 0 {
		return diags
	}

	sysStmt := findFirstKeyword(doc.Statements, "System")
	if sysStmt == nil {
		return diags
	}

	systemType, typeDiag := parseSystemType(*sysStmt)
	if typeDiag != nil {
		diags = append(diags, *typeDiag)
		return diags
	}

	keywordSet := statementKeywords(doc.Statements)

	switch systemType {
	case "LA":
		requireKeyword(&diags, keywordSet, "Frames", sysStmt.Line, sysStmt.Column)
		requireKeyword(&diags, keywordSet, "Connectors", sysStmt.Line, sysStmt.Column)
	case "CL":
		requireKeyword(&diags, keywordSet, "Setups", sysStmt.Line, sysStmt.Column)
	case "LS":
		for _, disallowed := range []string{"Frames", "Connectors", "Setups"} {
			if keywordSet[strings.ToLower(disallowed)] {
				diags = append(diags, Diagnostic{
					Severity: SeverityWarning,
					Message:  "unexpected \"" + disallowed + "\" block for LS system",
					Line:     sysStmt.Line,
					Column:   sysStmt.Column,
				})
			}
		}
	}

	return diags
}

// ValidateDataConstraints checks count declarations and basic references.
//
//nolint:gocyclo // Complexity is inherent to validating many different XGLL block types and constraints
func ValidateDataConstraints(doc *Document) []Diagnostic {
	var diags []Diagnostic
	if doc == nil || len(doc.Statements) == 0 {
		return diags
	}

	expect := newCountValidator()

	var (
		refSourceDefs   []refCheck
		refSources      []refCheck
		refFilterGroups []refCheck
		refBoxTypes     []refCheck
		refFrames       []refCheck
		refConnectors   []refCheck
	)

	boxTypes := make(map[string]struct{})
	frames := make(map[string]struct{})
	sources := make(map[string]struct{})
	sourceDefs := make(map[string]struct{})
	filterGroups := make(map[string]struct{})

	for _, stmt := range doc.Statements {
		switch stmt.Keyword {
		case "BoxTypes":
			expect.set("BoxType", stmt, &diags)
		case "BoxType":
			expect.see("BoxType", stmt, &diags)

			if name := argString(stmt, 1); name != "" {
				boxTypes[name] = struct{}{}
			}
		case "Sources":
			expect.set("Source", stmt, &diags)
		case "Source":
			expect.see("Source", stmt, &diags)

			if name := argString(stmt, 1); name != "" {
				sources[name] = struct{}{}
			}

			if ref := argString(stmt, 8); ref != "" {
				refSourceDefs = append(refSourceDefs, refCheck{Kind: "SourceDefinition", Name: ref, Stmt: stmt})
			}
		case "Input Configurations":
			expect.set("Input Configuration", stmt, &diags)
		case "Input Configuration":
			expect.see("Input Configuration", stmt, &diags)
		case "Inputs":
			expect.set("Input", stmt, &diags)
		case "Input":
			expect.see("Input", stmt, &diags)
		case "Links":
			expect.set("Link", stmt, &diags)
		case "Link":
			expect.see("Link", stmt, &diags)

			if ref := argString(stmt, 0); ref != "" {
				refSources = append(refSources, refCheck{Kind: "Source", Name: ref, Stmt: stmt})
			}

			if ref := argString(stmt, 1); ref != "" {
				refFilterGroups = append(refFilterGroups, refCheck{Kind: "FilterGroup", Name: ref, Stmt: stmt})
			}
		case "GeometryFiles":
			expect.set("GeometryFile", stmt, &diags)
		case "GeometryFile":
			expect.see("GeometryFile", stmt, &diags)
		case "Frames":
			expect.set("Frame", stmt, &diags)
		case "Frame":
			expect.see("Frame", stmt, &diags)

			if name := argString(stmt, 1); name != "" {
				frames[name] = struct{}{}
			}
		case "PinPoints":
			expect.set("PinPoint", stmt, &diags)
		case "PinPoint":
			expect.see("PinPoint", stmt, &diags)
		case "Connectors":
			expect.set("Connector", stmt, &diags)
		case "Connector":
			expect.see("Connector", stmt, &diags)

			if ref := argString(stmt, 0); ref != "" {
				refConnectors = append(refConnectors, refCheck{Kind: "Connector", Name: ref, Stmt: stmt})
			}

			if ref := argString(stmt, 1); ref != "" {
				refConnectors = append(refConnectors, refCheck{Kind: "Connector", Name: ref, Stmt: stmt})
			}
		case "Angles":
			expect.set("Angle", stmt, &diags)
		case "Angle":
			expect.see("Angle", stmt, &diags)
		case "Limits":
			expect.set("Limit", stmt, &diags)
		case "Limit":
			expect.see("Limit", stmt, &diags)

			if ref := argString(stmt, 1); ref != "" {
				refFrames = append(refFrames, refCheck{Kind: "Frame", Name: ref, Stmt: stmt})
			}

			if ref := argString(stmt, 2); ref != "" {
				refBoxTypes = append(refBoxTypes, refCheck{Kind: "BoxType", Name: ref, Stmt: stmt})
			}
		case "Warnings":
			expect.set("Warning", stmt, &diags)
		case "Warning":
			expect.see("Warning", stmt, &diags)

			if ref := argString(stmt, 2); ref != "" {
				refFrames = append(refFrames, refCheck{Kind: "Frame", Name: ref, Stmt: stmt})
			}
		case "SourceDefinitions":
			expect.set("SourceDefinition", stmt, &diags)
		case "SourceDefinition":
			expect.see("SourceDefinition", stmt, &diags)

			if name := argString(stmt, 0); name != "" {
				sourceDefs[name] = struct{}{}
			}
		case "FilterGroups":
			expect.set("FilterGroup", stmt, &diags)
		case "FilterGroup":
			expect.see("FilterGroup", stmt, &diags)

			if name := argString(stmt, 1); name != "" {
				filterGroups[name] = struct{}{}
			}
		case "FilterDefinitions":
			expect.set("FilterDefinition", stmt, &diags)
		case "FilterDefinition":
			expect.see("FilterDefinition", stmt, &diags)
		case "Setups":
			expect.set("Setup", stmt, &diags)
		case "Setup":
			expect.see("Setup", stmt, &diags)
		case "Boxes":
			expect.set("Box", stmt, &diags)
		case "Box":
			expect.see("Box", stmt, &diags)

			if ref := argString(stmt, 6); ref != "" {
				refBoxTypes = append(refBoxTypes, refCheck{Kind: "BoxType", Name: ref, Stmt: stmt})
			}
		}
	}

	expect.finish(&diags)

	resolveRefs(&diags, refSourceDefs, sourceDefs)
	resolveRefs(&diags, refSources, sources)
	resolveRefs(&diags, refFilterGroups, filterGroups)
	resolveRefs(&diags, refBoxTypes, boxTypes)
	resolveRefs(&diags, refFrames, frames)
	resolveConnectorRefs(&diags, refConnectors, boxTypes, frames)

	return diags
}

func parseSystemType(stmt Statement) (string, *Diagnostic) {
	if len(stmt.Args) < 3 {
		return "", &Diagnostic{
			Severity: SeverityError,
			Message:  "\"System\" expects 3 arguments (display, internal, type)",
			Line:     stmt.Line,
			Column:   stmt.Column,
		}
	}

	typeArg := stmt.Args[2]
	if typeArg.Kind != ValueString {
		return "", &Diagnostic{
			Severity: SeverityError,
			Message:  "\"System\" type must be a string",
			Line:     stmt.Line,
			Column:   stmt.Column,
		}
	}

	systemType := strings.ToUpper(typeArg.Str)
	switch systemType {
	case "LA", "CL", "LS":
		return systemType, nil
	default:
		return "", &Diagnostic{
			Severity: SeverityError,
			Message:  "unknown system type \"" + typeArg.Str + "\"",
			Line:     stmt.Line,
			Column:   stmt.Column,
		}
	}
}

func findFirstKeyword(statements []Statement, keyword string) *Statement {
	for i := range statements {
		if strings.EqualFold(statements[i].Keyword, keyword) {
			return &statements[i]
		}
	}

	return nil
}

func statementKeywords(statements []Statement) map[string]bool {
	set := make(map[string]bool, len(statements))
	for _, stmt := range statements {
		set[strings.ToLower(stmt.Keyword)] = true
	}

	return set
}

func requireKeyword(diags *[]Diagnostic, set map[string]bool, keyword string, line, col int) {
	if set[strings.ToLower(keyword)] {
		return
	}

	*diags = append(*diags, Diagnostic{
		Severity: SeverityError,
		Message:  "missing \"" + keyword + "\" block for system type",
		Line:     line,
		Column:   col,
	})
}

type countExpectation struct {
	count  int
	seen   int
	line   int
	column int
}

type countValidator struct {
	expect map[string]*countExpectation
}

func newCountValidator() *countValidator {
	return &countValidator{expect: make(map[string]*countExpectation)}
}

func (c *countValidator) set(target string, stmt Statement, diags *[]Diagnostic) {
	val, ok := argNumber(stmt, 0)
	if !ok {
		c.expect[target] = &countExpectation{
			count:  -1,
			line:   stmt.Line,
			column: stmt.Column,
		}

		return
	}

	c.finishTarget(target, diags)
	c.expect[target] = &countExpectation{
		count:  val,
		line:   stmt.Line,
		column: stmt.Column,
	}
}

func (c *countValidator) see(target string, stmt Statement, diags *[]Diagnostic) {
	entry := c.expect[target]
	if entry == nil {
		return
	}

	entry.seen++
	if entry.count >= 0 && entry.seen > entry.count {
		*diags = append(*diags, Diagnostic{
			Severity: SeverityError,
			Message:  "too many \"" + target + "\" entries (expected " + itoa(entry.count) + ")",
			Line:     stmt.Line,
			Column:   stmt.Column,
		})
	}
}

func (c *countValidator) finish(diags *[]Diagnostic) {
	for target := range c.expect {
		c.finishTarget(target, diags)
	}
}

func (c *countValidator) finishTarget(target string, diags *[]Diagnostic) {
	entry := c.expect[target]
	if entry == nil || entry.count < 0 {
		return
	}

	if entry.seen != entry.count && diags != nil {
		*diags = append(*diags, Diagnostic{
			Severity: SeverityError,
			Message:  "expected " + itoa(entry.count) + " \"" + target + "\" entries, got " + itoa(entry.seen),
			Line:     entry.line,
			Column:   entry.column,
		})
	}
}

func argNumber(stmt Statement, idx int) (int, bool) {
	if idx < 0 || idx >= len(stmt.Args) {
		return 0, false
	}

	arg := stmt.Args[idx]
	if arg.Kind != ValueNumber {
		return 0, false
	}

	n := int(arg.Num)
	if float64(n) != arg.Num {
		return 0, false
	}

	return n, true
}

func argString(stmt Statement, idx int) string {
	if idx < 0 || idx >= len(stmt.Args) {
		return ""
	}

	arg := stmt.Args[idx]
	if arg.Kind != ValueString {
		return ""
	}

	return strings.TrimSpace(arg.Str)
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

type refCheck struct {
	Kind string
	Name string
	Stmt Statement
}

func resolveRefs(diags *[]Diagnostic, refs []refCheck, set map[string]struct{}) {
	for _, ref := range refs {
		if ref.Name == "" {
			continue
		}

		if _, ok := set[ref.Name]; ok {
			continue
		}

		*diags = append(*diags, Diagnostic{
			Severity: SeverityWarning,
			Message:  ref.Kind + " reference \"" + ref.Name + "\" not found",
			Line:     ref.Stmt.Line,
			Column:   ref.Stmt.Column,
		})
	}
}

func resolveConnectorRefs(diags *[]Diagnostic, refs []refCheck, boxes, frames map[string]struct{}) {
	for _, ref := range refs {
		if ref.Name == "" {
			continue
		}

		if _, ok := boxes[ref.Name]; ok {
			continue
		}

		if _, ok := frames[ref.Name]; ok {
			continue
		}

		*diags = append(*diags, Diagnostic{
			Severity: SeverityWarning,
			Message:  "Connector reference \"" + ref.Name + "\" not found in BoxTypes or Frames",
			Line:     ref.Stmt.Line,
			Column:   ref.Stmt.Column,
		})
	}
}
