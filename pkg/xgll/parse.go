package xgll

import (
	"fmt"
	"io"
	"os"
)

// Parse reads an XGLL file from the provided reader.
func Parse(r io.Reader) (*Document, error) {
	// Read full input
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	// Parse into document
	p := newParser(string(data))
	doc := p.parse()

	// Fail if diagnostics contain errors
	if hasErrors(doc.Diagnostics) {
		return doc, fmt.Errorf("parse failed with %d errors", countErrors(doc.Diagnostics))
	}

	// Return parsed document
	return doc, nil
}

// ParseFile reads an XGLL file from disk.
func ParseFile(path string) (*Document, error) {
	// Open file and parse
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	return Parse(f)
}

func hasErrors(diags []Diagnostic) bool {
	// Check for error severity
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}

	return false
}

func countErrors(diags []Diagnostic) int {
	// Count error diagnostics
	count := 0

	for _, d := range diags {
		if d.Severity == SeverityError {
			count++
		}
	}

	return count
}
