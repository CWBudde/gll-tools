package xgll

import (
	"bufio"
	"fmt"
	"io"
)

type xgllTextWriter struct{}

func (w xgllTextWriter) Format() string {
	return "xgll"
}

func (w xgllTextWriter) Write(doc *Document, out io.Writer) error {
	return WriteXGLL(doc, out)
}

// WriteXGLL writes an XGLL document to the provided writer in text format.
func WriteXGLL(doc *Document, out io.Writer) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	bw := bufio.NewWriter(out)
	for i, stmt := range doc.Statements {
		if err := writeStatement(bw, stmt); err != nil {
			return err
		}
		if i < len(doc.Statements)-1 {
			if err := bw.WriteByte('\n'); err != nil {
				return err
			}
		}
	}

	return bw.Flush()
}

func writeStatement(w io.Writer, stmt Statement) error {
	if err := writeQuotedString(w, stmt.Keyword); err != nil {
		return err
	}

	for _, arg := range stmt.Args {
		if _, err := w.Write([]byte(", ")); err != nil {
			return err
		}

		switch arg.Kind {
		case ValueString:
			if err := writeQuotedString(w, escapeText(arg.Str)); err != nil {
				return err
			}
		case ValueNumber:
			if _, err := w.Write([]byte(formatNumber(arg.Num))); err != nil {
				return err
			}
		default:
			if _, err := w.Write([]byte(arg.Raw)); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeQuotedString(w io.Writer, value string) error {
	value = sanitizeXGLLString(value)
	if _, err := w.Write([]byte{'"'}); err != nil {
		return err
	}
	if _, err := w.Write([]byte(value)); err != nil {
		return err
	}
	if _, err := w.Write([]byte{'"'}); err != nil {
		return err
	}

	return nil
}

func sanitizeXGLLString(value string) string {
	if value == "" {
		return value
	}

	out := make([]byte, 0, len(value))
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch == '"' {
			out = append(out, '\'')
			continue
		}

		if ch < 32 || ch > 127 {
			out = append(out, '?')
			continue
		}

		out = append(out, ch)
	}

	return string(out)
}

func init() {
	RegisterWriter(xgllTextWriter{})
}
