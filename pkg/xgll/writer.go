package xgll

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// Writer emits a binary representation of an XGLL document.
type Writer interface {
	Format() string
	Write(doc *Document, w io.Writer) error
}

var (
	writerMu sync.RWMutex
	writers  = map[string]Writer{}
)

// RegisterWriter registers a writer by format.
func RegisterWriter(w Writer) {
	writerMu.Lock()
	defer writerMu.Unlock()

	writers[strings.ToLower(w.Format())] = w
}

// GetWriter returns a writer for the given format.
func GetWriter(format string) (Writer, error) {
	writerMu.RLock()
	defer writerMu.RUnlock()

	w, ok := writers[strings.ToLower(format)]
	if !ok {
		return nil, fmt.Errorf("unknown format %q", format)
	}

	return w, nil
}

// ListWriterFormats returns registered formats in sorted order.
func ListWriterFormats() []string {
	writerMu.RLock()
	defer writerMu.RUnlock()

	formats := make([]string, 0, len(writers))
	for k := range writers {
		formats = append(formats, k)
	}

	sort.Strings(formats)

	return formats
}
