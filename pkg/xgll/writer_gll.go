package xgll

import (
	"fmt"
	"io"

	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

type gllWriter struct{}

func (w gllWriter) Format() string {
	// Format key for registry
	return "gll"
}

func (w gllWriter) Write(doc *Document, out io.Writer) error {
	// Validate input document
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	// Build minimal GLL file from XGLL
	file, err := BuildGLLFile(doc)
	if err != nil {
		return err
	}

	// Encode to binary output
	enc := newGLLEncoder(out)

	return enc.writeFile(file)
}

func init() {
	// Register GLL writer
	RegisterWriter(gllWriter{})
}

// EncodeFile writes f to w in GLL binary format.
func EncodeFile(f *gllbin.File, w io.Writer) error {
	return newGLLEncoder(w).writeFile(f)
}
