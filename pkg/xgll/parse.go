package xgll

import (
	"fmt"
	"io"
	"os"
)

// Parse reads an XGLL file from the provided reader.
func Parse(r io.Reader) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	p := newParser(string(data))
	doc := p.parse()

	if hasErrors(doc.Diagnostics) {
		return doc, fmt.Errorf("parse failed with %d errors", countErrors(doc.Diagnostics))
	}

	return doc, nil
}

// ParseFile reads an XGLL file from disk.
func ParseFile(path string) (*Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	return Parse(f)
}

func hasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}

	return false
}

func countErrors(diags []Diagnostic) int {
	count := 0

	for _, d := range diags {
		if d.Severity == SeverityError {
			count++
		}
	}

	return count
}
