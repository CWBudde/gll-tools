package gll

import (
	"io"

	"github.com/cwbudde/gll-tools/internal/gll"
)

func readRawBlock(br *gll.ByteReader, start int64, size int) ([]byte, error) {
	if size <= 0 {
		return nil, nil
	}

	current := br.Offset()
	if _, err := br.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}

	data, err := br.ReadBytes(size)
	if err != nil {
		_, _ = br.Seek(current, io.SeekStart)
		return nil, err
	}

	if _, err := br.Seek(current, io.SeekStart); err != nil {
		return data, err
	}

	return data, nil
}
