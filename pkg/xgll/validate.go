package xgll

import (
	"strconv"
	"strings"
)

const (
	kwBoxType             = "BoxType"
	kwBoxTypes            = "BoxTypes"
	kwSourceDefinition    = "SourceDefinition"
	kwSourceDefinitions   = "SourceDefinitions"
	kwInputConfigurations = "Input Configurations"
	kwInputConfiguration  = "Input Configuration"
	kwInputs              = "Inputs"
	kwInput               = "Input"
	kwLinks               = "Links"
	kwLink                = "Link"
	kwFilterGroup         = "FilterGroup"
	kwFilterGroups        = "FilterGroups"
	kwFrame               = "Frame"
	kwConnector           = "Connector"
	kwConnectors          = "Connectors"
	kwFrames              = "Frames"
	kwSetups              = "Setups"
	kwLimit               = "Limit"
	kwLimits              = "Limits"
	kwWarning             = "Warning"
	kwWarnings            = "Warnings"
)

// ValidateSystemConstraints checks block presence and system type consistency.
func ValidateSystemConstraints(doc *Document) []Diagnostic {
	// Quick exit on empty input
	var diags []Diagnostic
	if doc == nil || len(doc.Statements) == 0 {
		return diags
	}

	// Locate System statement
	sysStmt := findFirstKeyword(doc.Statements, "System")
	if sysStmt == nil {
		return diags
	}

	// Parse system type
	systemType, typeDiag := parseSystemType(*sysStmt)
	if typeDiag != nil {
		diags = append(diags, *typeDiag)
		return diags
	}

	// Build keyword set for quick lookup
	keywordSet := statementKeywords(doc.Statements)

	// Validate block requirements by system type
	switch systemType {
	case "LA":
		requireKeyword(&diags, keywordSet, kwFrames, sysStmt.Line, sysStmt.Column)
		requireKeyword(&diags, keywordSet, kwConnectors, sysStmt.Line, sysStmt.Column)
	case "CL":
		requireKeyword(&diags, keywordSet, kwSetups, sysStmt.Line, sysStmt.Column)
	case "LS":
		// LS must not contain LA/CL-specific blocks
		for _, disallowed := range []string{kwFrames, kwConnectors, kwSetups} {
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

	// Return diagnostics
	return diags
}

// ValidateDataConstraints checks count declarations and basic references.
//
//nolint:gocyclo // Complexity is inherent to validating many different XGLL block types and constraints
func ValidateDataConstraints(doc *Document) []Diagnostic {
	// Quick exit on empty input
	var diags []Diagnostic
	if doc == nil || len(doc.Statements) == 0 {
		return diags
	}

	// Count validator for declaration blocks
	expect := newCountValidator()

	// Reference checks collected during scan
	var (
		refSourceDefs   []refCheck
		refSources      []refCheck
		refFilterGroups []refCheck
		refBoxTypes     []refCheck
		refFrames       []refCheck
		refConnectors   []refCheck
	)

	// Index sets for name resolution
	boxTypes := make(map[string]struct{})
	frames := make(map[string]struct{})
	sources := make(map[string]struct{})
	sourceDefs := make(map[string]struct{})
	filterGroups := make(map[string]struct{})

	// Walk statements and validate counts/references
	for _, stmt := range doc.Statements {
		switch stmt.Keyword {
		case kwBoxTypes:
			// Declared count for BoxType
			expect.set(kwBoxType, stmt, &diags)
		case kwBoxType:
			// Count BoxType entry
			expect.see(kwBoxType, stmt, &diags)

			// Collect BoxType names
			if name := argString(stmt, 1); name != "" {
				boxTypes[name] = struct{}{}
			}
		case "Sources":
			// Declared count for Source
			expect.set("Source", stmt, &diags)
		case "Source":
			// Count Source entry
			expect.see("Source", stmt, &diags)

			// Collect source names
			if name := argString(stmt, 1); name != "" {
				sources[name] = struct{}{}
			}

			// Capture SourceDefinition reference
			if ref := argString(stmt, 8); ref != "" {
				refSourceDefs = append(refSourceDefs, refCheck{Kind: kwSourceDefinition, Name: ref, Stmt: stmt})
			}
		case kwInputConfigurations:
			// Declared count for Input Configuration
			expect.set(kwInputConfiguration, stmt, &diags)
		case kwInputConfiguration:
			// Count Input Configuration entry
			expect.see(kwInputConfiguration, stmt, &diags)
		case kwInputs:
			// Declared count for Input
			expect.set(kwInput, stmt, &diags)
		case kwInput:
			// Count Input entry
			expect.see(kwInput, stmt, &diags)
		case kwLinks:
			// Declared count for Link
			expect.set(kwLink, stmt, &diags)
		case kwLink:
			// Count Link entry
			expect.see(kwLink, stmt, &diags)

			// Capture Link references
			if ref := argString(stmt, 0); ref != "" {
				refSources = append(refSources, refCheck{Kind: "Source", Name: ref, Stmt: stmt})
			}

			if ref := argString(stmt, 1); ref != "" {
				refFilterGroups = append(refFilterGroups, refCheck{Kind: kwFilterGroup, Name: ref, Stmt: stmt})
			}
		case "GeometryFiles":
			// Declared count for GeometryFile
			expect.set("GeometryFile", stmt, &diags)
		case "GeometryFile":
			// Count GeometryFile entry
			expect.see("GeometryFile", stmt, &diags)
		case kwFrames:
			// Declared count for Frame
			expect.set(kwFrame, stmt, &diags)
		case kwFrame:
			// Count Frame entry
			expect.see(kwFrame, stmt, &diags)

			// Collect frame names
			if name := argString(stmt, 1); name != "" {
				frames[name] = struct{}{}
			}
		case "PinPoints":
			// Declared count for PinPoint
			expect.set("PinPoint", stmt, &diags)
		case "PinPoint":
			// Count PinPoint entry
			expect.see("PinPoint", stmt, &diags)
		case kwConnectors:
			// Declared count for Connector
			expect.set(kwConnector, stmt, &diags)
		case kwConnector:
			// Count Connector entry
			expect.see(kwConnector, stmt, &diags)

			// Capture connector references
			if ref := argString(stmt, 0); ref != "" {
				refConnectors = append(refConnectors, refCheck{Kind: kwConnector, Name: ref, Stmt: stmt})
			}

			if ref := argString(stmt, 1); ref != "" {
				refConnectors = append(refConnectors, refCheck{Kind: kwConnector, Name: ref, Stmt: stmt})
			}
		case "Angles":
			// Declared count for Angle
			expect.set("Angle", stmt, &diags)
		case "Angle":
			// Count Angle entry
			expect.see("Angle", stmt, &diags)
		case kwLimits:
			// Declared count for Limit
			expect.set(kwLimit, stmt, &diags)
		case kwLimit:
			// Count Limit entry
			expect.see(kwLimit, stmt, &diags)

			// Capture frame reference
			if ref := argString(stmt, 1); ref != "" {
				refFrames = append(refFrames, refCheck{Kind: kwFrame, Name: ref, Stmt: stmt})
			}

			// Capture box type reference
			if ref := argString(stmt, 2); ref != "" {
				refBoxTypes = append(refBoxTypes, refCheck{Kind: kwBoxType, Name: ref, Stmt: stmt})
			}
		case kwWarnings:
			// Declared count for Warning
			expect.set(kwWarning, stmt, &diags)
		case kwWarning:
			// Count Warning entry
			expect.see(kwWarning, stmt, &diags)

			// Capture frame reference
			if ref := argString(stmt, 2); ref != "" {
				refFrames = append(refFrames, refCheck{Kind: kwFrame, Name: ref, Stmt: stmt})
			}
		case kwSourceDefinitions:
			// Declared count for SourceDefinition
			expect.set(kwSourceDefinition, stmt, &diags)
		case kwSourceDefinition:
			// Count SourceDefinition entry
			expect.see(kwSourceDefinition, stmt, &diags)

			// Collect SourceDefinition names
			if name := argString(stmt, 0); name != "" {
				sourceDefs[name] = struct{}{}
			}
		case kwFilterGroups:
			// Declared count for FilterGroup
			expect.set(kwFilterGroup, stmt, &diags)
		case kwFilterGroup:
			// Count FilterGroup entry
			expect.see(kwFilterGroup, stmt, &diags)

			// Collect FilterGroup names
			if name := argString(stmt, 1); name != "" {
				filterGroups[name] = struct{}{}
			}
		case "FilterDefinitions":
			// Declared count for FilterDefinition
			expect.set("FilterDefinition", stmt, &diags)
		case "FilterDefinition":
			// Count FilterDefinition entry
			expect.see("FilterDefinition", stmt, &diags)
		case kwSetups:
			// Declared count for Setup
			expect.set("Setup", stmt, &diags)
		case "Setup":
			// Count Setup entry
			expect.see("Setup", stmt, &diags)
		case "Boxes":
			// Declared count for Box
			expect.set("Box", stmt, &diags)
		case "Box":
			// Count Box entry
			expect.see("Box", stmt, &diags)

			// Capture BoxType reference
			if ref := argString(stmt, 6); ref != "" {
				refBoxTypes = append(refBoxTypes, refCheck{Kind: kwBoxType, Name: ref, Stmt: stmt})
			}
		}
	}

	// Finalize expected counts
	expect.finish(&diags)

	// Resolve collected references
	resolveRefs(&diags, refSourceDefs, sourceDefs)
	resolveRefs(&diags, refSources, sources)
	resolveRefs(&diags, refFilterGroups, filterGroups)
	resolveRefs(&diags, refBoxTypes, boxTypes)
	resolveRefs(&diags, refFrames, frames)
	resolveConnectorRefs(&diags, refConnectors, boxTypes, frames)

	// Return diagnostics
	return diags
}

func parseSystemType(stmt Statement) (string, *Diagnostic) {
	// Ensure System has required arguments
	if len(stmt.Args) < 3 {
		return "", &Diagnostic{
			Severity: SeverityError,
			Message:  "\"System\" expects 3 arguments (display, internal, type)",
			Line:     stmt.Line,
			Column:   stmt.Column,
		}
	}

	// Ensure type argument is a string
	typeArg := stmt.Args[2]
	if typeArg.Kind != ValueString {
		return "", &Diagnostic{
			Severity: SeverityError,
			Message:  "\"System\" type must be a string",
			Line:     stmt.Line,
			Column:   stmt.Column,
		}
	}

	// Normalize system type
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
	// Find first keyword match (case-insensitive)
	for i := range statements {
		if strings.EqualFold(statements[i].Keyword, keyword) {
			return &statements[i]
		}
	}

	return nil
}

func statementKeywords(statements []Statement) map[string]bool {
	// Build set of statement keywords
	set := make(map[string]bool, len(statements))
	for _, stmt := range statements {
		set[strings.ToLower(stmt.Keyword)] = true
	}

	return set
}

func requireKeyword(diags *[]Diagnostic, set map[string]bool, keyword string, line, col int) {
	// Require presence of keyword
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
	// Expectation for count blocks
	count  int
	seen   int
	line   int
	column int
}

type countValidator struct {
	// Tracks expected counts by target
	expect map[string]*countExpectation
}

func newCountValidator() *countValidator {
	// Create empty validator
	return &countValidator{expect: make(map[string]*countExpectation)}
}

func (c *countValidator) set(target string, stmt Statement, diags *[]Diagnostic) {
	// Set expected count for target
	val, ok := argNumber(stmt, 0)
	if !ok {
		c.expect[target] = &countExpectation{
			count:  -1,
			line:   stmt.Line,
			column: stmt.Column,
		}

		return
	}

	// Finalize previous expectation
	c.finishTarget(target, diags)
	c.expect[target] = &countExpectation{
		count:  val,
		line:   stmt.Line,
		column: stmt.Column,
	}
}

func (c *countValidator) see(target string, stmt Statement, diags *[]Diagnostic) {
	// Increment seen count
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
	// Finalize all targets
	for target := range c.expect {
		c.finishTarget(target, diags)
	}
}

func (c *countValidator) finishTarget(target string, diags *[]Diagnostic) {
	// Emit diagnostic if counts mismatch
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
	// Extract integer argument
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
	// Extract string argument
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
	// Int to string helper
	return strconv.Itoa(n)
}

type refCheck struct {
	// Reference to validate
	Kind string
	Name string
	Stmt Statement
}

func resolveRefs(diags *[]Diagnostic, refs []refCheck, set map[string]struct{}) {
	// Resolve generic references
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
	// Resolve connector references against box/frame sets
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
