package xgll

import (
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	lex        *lexer
	lookahead  token
	consumed   bool
	statements []Statement
	diags      []Diagnostic
}

func newParser(input string) *parser {
	// Initialize lexer and parser
	lex := newLexer(input)
	return &parser{lex: lex}
}

func (p *parser) parse() *Document {
	// Parse tokens into statements
	for {
		stmt, ok := p.parseStatement()
		if !ok {
			break
		}

		if stmt.Keyword == "" && len(stmt.Args) == 0 {
			continue
		}

		p.statements = append(p.statements, stmt)
	}

	// Merge lexer diagnostics
	p.diags = append(p.diags, p.lex.diags...)

	// Build document with blocks and validation
	doc := &Document{
		Statements:  p.statements,
		Diagnostics: p.diags,
	}
	doc.Blocks = buildBlocks(doc.Statements)
	doc.Diagnostics = append(doc.Diagnostics, validateBlocks(doc)...)

	return doc
}

func (p *parser) parseStatement() (Statement, bool) {
	// Consume tokens until a statement is formed
	for {
		tok := p.next()
		switch tok.Kind {
		case tokenEOF:
			return Statement{}, false
		case tokenNewline:
			continue
		case tokenInvalid:
			return Statement{}, true
		case tokenString:
			stmt := Statement{
				Keyword: tok.Value,
				Line:    tok.Line,
				Column:  tok.Column,
			}
			stmt.IsComment = tok.Value == ";"
			// Parse argument list
			args := p.parseArgs()
			stmt.Args = args

			p.consumeLineEnd()

			return stmt, true
		default:
			// Non-quoted keyword is invalid
			p.diag(tok, "expected quoted keyword")
			p.consumeLineEnd()

			return Statement{}, true
		}
	}
}

func (p *parser) parseArgs() []Value {
	// Parse comma-separated values
	var args []Value

	expectValue := true

	for {
		tok := p.peek()
		switch tok.Kind {
		case tokenEOF, tokenNewline:
			return args
		case tokenComma:
			p.next()

			expectValue = true

			continue
		case tokenString:
			if !expectValue && len(args) > 0 {
				p.diag(tok, "missing comma before string value")
			}

			p.next()

			args = append(args, Value{
				Kind: ValueString,
				Raw:  tok.Value,
				Str:  tok.Value,
			})
			expectValue = false
		case tokenNumber:
			if !expectValue && len(args) > 0 {
				p.diag(tok, "missing comma before number value")
			}

			p.next()

			num, _ := strconv.ParseFloat(tok.Value, 64)
			args = append(args, Value{
				Kind: ValueNumber,
				Raw:  tok.Value,
				Num:  num,
			})
			expectValue = false
		case tokenInvalid:
			p.next()

			expectValue = false
		default:
			// Unexpected token inside args
			p.diag(tok, fmt.Sprintf("unexpected token %q", tok.Value))
			p.next()

			expectValue = false
		}
	}
}

func (p *parser) consumeLineEnd() {
	// Consume tokens until newline/EOF
	for {
		tok := p.peek()
		if tok.Kind == tokenNewline {
			p.next()
			return
		}

		if tok.Kind == tokenEOF {
			return
		}

		p.next()
	}
}

func (p *parser) next() token {
	// Read next token (honor lookahead)
	if p.consumed {
		p.consumed = false
		return p.lookahead
	}

	p.lookahead = p.lex.nextToken()

	return p.lookahead
}

func (p *parser) peek() token {
	// Peek next token without consuming
	if !p.consumed {
		p.lookahead = p.lex.nextToken()
		p.consumed = true
	}

	return p.lookahead
}

func (p *parser) diag(tok token, msg string) {
	// Append parser diagnostic
	p.diags = append(p.diags, Diagnostic{
		Severity: SeverityError,
		Message:  msg,
		Line:     tok.Line,
		Column:   tok.Column,
	})
}

func buildBlocks(statements []Statement) []Block {
	// Partition statements into header/system/layout/data blocks
	var blocks []Block
	if len(statements) == 0 {
		return blocks
	}

	layoutIdx := indexOfKeyword(statements, "Layout")
	dataIdx := indexOfKeyword(statements, "Data")
	systemIdx := indexOfKeyword(statements, "System")

	appendBlock := func(typ BlockType, start, end int) {
		// Append slice if bounds are valid
		if start < 0 || end < 0 || start >= end || start >= len(statements) {
			return
		}

		if end > len(statements) {
			end = len(statements)
		}

		blocks = append(blocks, Block{
			Type:       typ,
			Statements: statements[start:end],
		})
	}

	headerEnd := len(statements)
	if systemIdx >= 0 {
		headerEnd = systemIdx
	}

	appendBlock(BlockHeader, 0, headerEnd)

	if systemIdx >= 0 {
		// System block up to next layout/data
		systemEnd := len(statements)
		if layoutIdx >= 0 && layoutIdx > systemIdx {
			systemEnd = layoutIdx
		} else if dataIdx >= 0 && dataIdx > systemIdx {
			systemEnd = dataIdx
		}

		appendBlock(BlockSystem, systemIdx, systemEnd)
	}

	if layoutIdx >= 0 {
		// Layout block up to data
		layoutEnd := len(statements)
		if dataIdx >= 0 && dataIdx > layoutIdx {
			layoutEnd = dataIdx
		}

		appendBlock(BlockLayout, layoutIdx, layoutEnd)
	}

	if dataIdx >= 0 {
		// Data block to end
		appendBlock(BlockData, dataIdx, len(statements))
	}

	if len(blocks) == 0 {
		appendBlock(BlockOther, 0, len(statements))
	}

	return blocks
}

func indexOfKeyword(statements []Statement, keyword string) int {
	// Case-insensitive lookup of keyword
	for i, stmt := range statements {
		if strings.EqualFold(stmt.Keyword, keyword) {
			return i
		}
	}

	return -1
}

func validateBlocks(doc *Document) []Diagnostic {
	// Validate presence and ordering of key blocks
	var diags []Diagnostic
	if len(doc.Statements) == 0 {
		return diags
	}

	first := doc.Statements[0]
	if first.Keyword != "GLL" {
		// First line must be GLL signature
		diags = append(diags, Diagnostic{
			Severity: SeverityError,
			Message:  "file must start with \"GLL\"",
			Line:     first.Line,
			Column:   first.Column,
		})
	}

	hasFormat := false
	hasFormatVersion := false

	formatIdx, versionIdx, systemIdx, layoutIdx, dataIdx := -1, -1, -1, -1, -1

	for i, stmt := range doc.Statements {
		// Track indices of key statements
		switch stmt.Keyword {
		case "Format":
			hasFormat = true
			formatIdx = i
		case "FormatVersion":
			hasFormatVersion = true
			versionIdx = i
		case "System":
			systemIdx = i
		case "Layout":
			layoutIdx = i
		case "Data":
			dataIdx = i
		}
	}

	if !hasFormat {
		// Required header line
		diags = append(diags, Diagnostic{Severity: SeverityError, Message: "missing \"Format\" line"})
	}

	if !hasFormatVersion {
		// Required header line
		diags = append(diags, Diagnostic{Severity: SeverityError, Message: "missing \"FormatVersion\" line"})
	}

	if systemIdx < 0 {
		// Required system block
		diags = append(diags, Diagnostic{Severity: SeverityError, Message: "missing \"System\" line"})
	}

	if layoutIdx < 0 {
		// Optional but recommended block
		diags = append(diags, Diagnostic{Severity: SeverityWarning, Message: "missing \"Layout\" block"})
	}

	if dataIdx < 0 {
		// Required data block
		diags = append(diags, Diagnostic{Severity: SeverityError, Message: "missing \"Data\" block"})
	}

	if formatIdx >= 0 && systemIdx >= 0 && formatIdx > systemIdx {
		// Enforce ordering
		diags = append(diags, Diagnostic{Severity: SeverityError, Message: "\"Format\" must precede \"System\""})
	}

	if versionIdx >= 0 && systemIdx >= 0 && versionIdx > systemIdx {
		// Enforce ordering
		diags = append(diags, Diagnostic{Severity: SeverityError, Message: "\"FormatVersion\" must precede \"System\""})
	}

	if layoutIdx >= 0 && dataIdx >= 0 && layoutIdx > dataIdx {
		// Enforce ordering
		diags = append(diags, Diagnostic{Severity: SeverityError, Message: "\"Layout\" must precede \"Data\""})
	}

	return diags
}
