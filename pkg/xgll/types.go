package xgll

// ValueKind represents a typed value in an XGLL statement.
type ValueKind int

const (
	// Supported value kinds
	ValueString ValueKind = iota
	ValueNumber
)

// Value is a parsed argument value for a statement.
type Value struct {
	// Parsed value metadata
	Kind ValueKind
	Raw  string
	Str  string
	Num  float64
}

// Statement is a single XGLL line (keyword + arguments).
type Statement struct {
	// Statement data and location
	Keyword   string
	Args      []Value
	Line      int
	Column    int
	IsComment bool
}

// BlockType is a high-level grouping of statements.
type BlockType string

const (
	// Block identifiers for validation
	BlockHeader BlockType = "Header"
	BlockSystem BlockType = "SystemHeader"
	BlockLayout BlockType = "Layout"
	BlockData   BlockType = "Data"
	BlockOther  BlockType = "Other"
)

// Block groups contiguous statements into a higher-level section.
type Block struct {
	// Block content and type
	Type       BlockType
	Statements []Statement
}

// Document is the parsed XGLL file.
type Document struct {
	// Parsed content and diagnostics
	Statements  []Statement
	Blocks      []Block
	Diagnostics []Diagnostic
}

// DiagnosticSeverity indicates the severity of a parser message.
type DiagnosticSeverity int

const (
	// Warning and error severities
	SeverityWarning DiagnosticSeverity = iota
	SeverityError
)

// Diagnostic provides parser error context.
type Diagnostic struct {
	// Diagnostic message and location
	Severity DiagnosticSeverity
	Message  string
	Line     int
	Column   int
}
