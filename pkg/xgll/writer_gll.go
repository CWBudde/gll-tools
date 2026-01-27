package xgll

import (
	"fmt"
	"io"
)

type gllWriter struct{}

func (w gllWriter) Format() string {
	return "gll"
}

func (w gllWriter) Write(doc *Document, out io.Writer) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	file, err := BuildGLLFile(doc)
	if err != nil {
		return err
	}

	enc := newGLLEncoder(out)

	return enc.writeFile(file)
}

func init() {
	RegisterWriter(gllWriter{})
}
