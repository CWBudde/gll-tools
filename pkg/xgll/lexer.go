package xgll

import (
	"fmt"
	"strconv"
)

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenString
	tokenNumber
	tokenComma
	tokenNewline
	tokenInvalid
)

type token struct {
	Kind   tokenKind
	Value  string
	Line   int
	Column int
}

type lexer struct {
	input []rune
	pos   int
	line  int
	col   int
	diags []Diagnostic
}

func newLexer(input string) *lexer {
	return &lexer{
		input: []rune(input),
		line:  1,
		col:   1,
	}
}

func (l *lexer) nextToken() token {
	for {
		if l.pos >= len(l.input) {
			return token{Kind: tokenEOF, Line: l.line, Column: l.col}
		}

		ch := l.peek()

		if ch == '\r' {
			l.advance()

			if l.peek() == '\n' {
				l.advance()
			}

			l.line++
			l.col = 1

			return token{Kind: tokenNewline, Line: l.line - 1, Column: 1}
		}

		if ch == '\n' {
			l.advance()
			l.line++
			l.col = 1

			return token{Kind: tokenNewline, Line: l.line - 1, Column: 1}
		}

		if ch == ' ' || ch == '\t' {
			l.advance()
			continue
		}

		if ch < 32 || ch > 127 {
			l.diag(SeverityError, fmt.Sprintf("invalid character %q", ch))
			l.advance()

			return token{Kind: tokenInvalid, Line: l.line, Column: l.col - 1}
		}

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
	startLine := l.line
	startCol := l.col
	start := l.pos

	if l.peek() == '+' || l.peek() == '-' {
		l.advance()
	}

	for isDigit(l.peek()) {
		l.advance()
	}

	if l.peek() == '.' {
		l.advance()

		for isDigit(l.peek()) {
			l.advance()
		}
	}

	if l.peek() == 'e' || l.peek() == 'E' {
		l.advance()

		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}

		for isDigit(l.peek()) {
			l.advance()
		}
	}

	value := string(l.input[start:l.pos])
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		l.diagAt(SeverityError, fmt.Sprintf("invalid number %q", value), startLine, startCol)
		return token{Kind: tokenInvalid, Line: startLine, Column: startCol}
	}

	return token{Kind: tokenNumber, Value: value, Line: startLine, Column: startCol}
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}

	return l.input[l.pos]
}

func (l *lexer) peekNext() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}

	return l.input[l.pos+1]
}

func (l *lexer) advance() {
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
	l.diags = append(l.diags, Diagnostic{
		Severity: sev,
		Message:  msg,
		Line:     l.line,
		Column:   l.col,
	})
}

func (l *lexer) diagAt(sev DiagnosticSeverity, msg string, line, col int) {
	l.diags = append(l.diags, Diagnostic{
		Severity: sev,
		Message:  msg,
		Line:     line,
		Column:   col,
	})
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isNumberStart(ch rune, next rune) bool {
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
