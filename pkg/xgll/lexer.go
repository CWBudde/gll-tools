package xgll

import (
	"fmt"
	"strconv"
)

type tokenKind int

const (
	// Token kinds for XGLL lexer
	tokenEOF tokenKind = iota
	tokenString
	tokenNumber
	tokenComma
	tokenNewline
	tokenInvalid
)

type token struct {
	// Token metadata
	Kind   tokenKind
	Value  string
	Line   int
	Column int
}

type lexer struct {
	// Lexer state and diagnostics
	input []rune
	pos   int
	line  int
	col   int
	diags []Diagnostic
}

func newLexer(input string) *lexer {
	// Initialize lexer with input
	return &lexer{
		input: []rune(input),
		line:  1,
		col:   1,
	}
}

func (l *lexer) nextToken() token {
	// Main tokenization loop
	for {
		if l.pos >= len(l.input) {
			return token{Kind: tokenEOF, Line: l.line, Column: l.col}
		}

		// Inspect next rune
		ch := l.peek()

		// Normalize CRLF or CR newlines
		if ch == '\r' {
			l.advance()

			if l.peek() == '\n' {
				l.advance()
			}

			l.line++
			l.col = 1

			return token{Kind: tokenNewline, Line: l.line - 1, Column: 1}
		}

		// Handle LF newline
		if ch == '\n' {
			l.advance()
			l.line++
			l.col = 1

			return token{Kind: tokenNewline, Line: l.line - 1, Column: 1}
		}

		// Skip whitespace
		if ch == ' ' || ch == '\t' {
			l.advance()
			continue
		}

		// Reject non-ASCII control chars
		if ch < 32 || ch > 127 {
			l.diag(SeverityError, fmt.Sprintf("invalid character %q", ch))
			l.advance()

			return token{Kind: tokenInvalid, Line: l.line, Column: l.col - 1}
		}

		// Tokenize by leading character
		switch ch {
		case '"':
			return l.readString()
		case ',':
			l.advance()
			return token{Kind: tokenComma, Value: ",", Line: l.line, Column: l.col - 1}
		default:
			if isNumberStart(ch, l.peekNext()) {
				return l.readNumber()
			}

			l.diag(SeverityError, fmt.Sprintf("unexpected character %q", ch))
			l.advance()

			return token{Kind: tokenInvalid, Line: l.line, Column: l.col - 1}
		}
	}
}

func (l *lexer) readString() token {
	// Parse quoted string literal
	startLine := l.line
	startCol := l.col
	l.advance()

	start := l.pos
	for {
		if l.pos >= len(l.input) {
			l.diagAt(SeverityError, "unterminated string", startLine, startCol)
			return token{Kind: tokenInvalid, Line: startLine, Column: startCol}
		}

		ch := l.peek()
		if ch == '"' {
			value := string(l.input[start:l.pos])
			l.advance()

			return token{Kind: tokenString, Value: value, Line: startLine, Column: startCol}
		}

		if ch == '\r' || ch == '\n' {
			l.diagAt(SeverityError, "unterminated string", startLine, startCol)
			return token{Kind: tokenInvalid, Line: startLine, Column: startCol}
		}

		if ch < 32 || ch > 127 {
			l.diag(SeverityError, fmt.Sprintf("invalid character %q in string", ch))
		}

		l.advance()
	}
}

func (l *lexer) readNumber() token {
	// Parse numeric literal
	startLine := l.line
	startCol := l.col
	start := l.pos

	// Optional sign
	if l.peek() == '+' || l.peek() == '-' {
		l.advance()
	}

	// Integer part
	for isDigit(l.peek()) {
		l.advance()
	}

	// Fractional part
	if l.peek() == '.' {
		l.advance()

		for isDigit(l.peek()) {
			l.advance()
		}
	}

	// Exponent part
	if l.peek() == 'e' || l.peek() == 'E' {
		l.advance()

		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}

		for isDigit(l.peek()) {
			l.advance()
		}
	}

	// Validate numeric value
	value := string(l.input[start:l.pos])
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		l.diagAt(SeverityError, fmt.Sprintf("invalid number %q", value), startLine, startCol)
		return token{Kind: tokenInvalid, Line: startLine, Column: startCol}
	}

	return token{Kind: tokenNumber, Value: value, Line: startLine, Column: startCol}
}

func (l *lexer) peek() rune {
	// Peek current rune
	if l.pos >= len(l.input) {
		return 0
	}

	return l.input[l.pos]
}

func (l *lexer) peekNext() rune {
	// Peek next rune
	if l.pos+1 >= len(l.input) {
		return 0
	}

	return l.input[l.pos+1]
}

func (l *lexer) advance() {
	// Advance cursor and update line/column
	if l.pos < len(l.input) {
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}

		l.pos++
	}
}

func (l *lexer) diag(sev DiagnosticSeverity, msg string) {
	// Record diagnostic at current position
	l.diags = append(l.diags, Diagnostic{
		Severity: sev,
		Message:  msg,
		Line:     l.line,
		Column:   l.col,
	})
}

func (l *lexer) diagAt(sev DiagnosticSeverity, msg string, line, col int) {
	// Record diagnostic at specified position
	l.diags = append(l.diags, Diagnostic{
		Severity: sev,
		Message:  msg,
		Line:     line,
		Column:   col,
	})
}

func isDigit(ch rune) bool {
	// ASCII digit check
	return ch >= '0' && ch <= '9'
}

func isNumberStart(ch rune, next rune) bool {
	// Detect number prefix (+, -, ., digit)
	if isDigit(ch) {
		return true
	}

	if (ch == '+' || ch == '-') && isDigit(next) {
		return true
	}

	if ch == '.' && isDigit(next) {
		return true
	}

	return false
}
