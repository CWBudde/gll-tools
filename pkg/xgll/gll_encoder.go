package xgll

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	internalgll "github.com/cwbudde/gll-tools/internal/gll"
	gllbin "github.com/cwbudde/gll-tools/pkg/gll"
)

type gllEncoder struct {
	w io.Writer
}

func newGLLEncoder(w io.Writer) *gllEncoder {
	return &gllEncoder{w: w}
}

func (e *gllEncoder) writeFile(file *gllbin.File) error {
	if file == nil {
		return fmt.Errorf("file is nil")
	}

	if err := e.writeHeader(file.Header); err != nil {
		return err
	}

	if len(file.GenSystem.RawBlock) > 0 {
		if _, err := e.w.Write(file.GenSystem.RawBlock); err != nil {
			return fmt.Errorf("write gensystem raw: %w", err)
		}
	} else {
		genSystem, err := e.encodeGenSystem(file)
		if err != nil {
			return err
		}

		if _, err := e.w.Write(genSystem); err != nil {
			return fmt.Errorf("write gensystem: %w", err)
		}
	}

	if len(file.RawTail) > 0 {
		if _, err := e.w.Write(file.RawTail); err != nil {
			return fmt.Errorf("write tail: %w", err)
		}
	}

	return nil
}

func (e *gllEncoder) writeHeader(header gllbin.Header) error {
	if _, err := e.w.Write([]byte(internalgll.MagicEGLL)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}

	if err := binary.Write(e.w, binary.LittleEndian, int32(0)); err != nil {
		return fmt.Errorf("write reserved: %w", err)
	}

	if err := writeString(e.w, internalgll.MagicEASEGLL); err != nil {
		return fmt.Errorf("write format id: %w", err)
	}

	version := header.FormatVersion
	if version == 0 {
		version = defaultBinaryFormatVersion
	}

	if err := binary.Write(e.w, binary.LittleEndian, version); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	if err := binary.Write(e.w, binary.LittleEndian, header.SubVersion); err != nil {
		return fmt.Errorf("write sub-version: %w", err)
	}

	if version >= 4 {
		if _, err := e.w.Write(header.Checksum[:]); err != nil {
			return fmt.Errorf("write checksum: %w", err)
		}
	}

	if version >= 6 {
		hashLen := int32(0)
		if hasHash(header.HashID) {
			hashLen = int32(len(header.HashID))
		}
		if err := binary.Write(e.w, binary.LittleEndian, hashLen); err != nil {
			return fmt.Errorf("write hash length: %w", err)
		}

		if hashLen > 0 {
			if _, err := e.w.Write(header.HashID[:]); err != nil {
				return fmt.Errorf("write hash: %w", err)
			}
		}
	}

	return nil
}

func (e *gllEncoder) encodeGenSystem(file *gllbin.File) ([]byte, error) {
	sys := file.GenSystem
	var payload bytes.Buffer

	if err := writeString(&payload, sys.Label); err != nil {
		return nil, err
	}

	if err := binary.Write(&payload, binary.LittleEndian, sys.Version); err != nil {
		return nil, fmt.Errorf("write version: %w", err)
	}

	if err := writeString(&payload, sys.Key); err != nil {
		return nil, err
	}

	if err := binary.Write(&payload, binary.LittleEndian, int32(sys.Type)); err != nil {
		return nil, fmt.Errorf("write type: %w", err)
	}

	if err := writeString(&payload, sys.Company); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.InfoText); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.CopyrightText); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.SupportText); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.WebsiteText); err != nil {
		return nil, err
	}

	if err := writeString(&payload, sys.EmailText); err != nil {
		return nil, err
	}

	if err := binary.Write(&payload, binary.LittleEndian, sys.BackgroundColor); err != nil {
		return nil, fmt.Errorf("write background color: %w", err)
	}

	db, err := e.encodeDatabase(file.Database)
	if err != nil {
		return nil, err
	}

	if _, err := payload.Write(db); err != nil {
		return nil, fmt.Errorf("write database: %w", err)
	}

	if sys.FlagsPresent {
		flags := int32(0)
		if sys.AllowUserDefinedClusterSetup {
			flags |= 0x01
		}
		if sys.EnableForSubArrays {
			flags |= 0x02
		}

		if err := binary.Write(&payload, binary.LittleEndian, flags); err != nil {
			return nil, fmt.Errorf("write gensystem flags: %w", err)
		}
	}

	return encodeBlock(sys.SubVersion, payload.Bytes()), nil
}

func (e *gllEncoder) encodeDatabase(db *gllbin.Database) ([]byte, error) {
	if db != nil && len(db.RawBlock) > 0 {
		return db.RawBlock, nil
	}

	var payload bytes.Buffer

	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write db field1: %w", err)
	}

	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write db field2: %w", err)
	}

	// DataFiles count
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write datafiles count: %w", err)
	}

	// BoxTypes buffer (empty block)
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write boxtypes block: %w", err)
	}

	// Frames buffer (empty block)
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write frames block: %w", err)
	}

	// Connectors buffer (empty block)
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write connectors block: %w", err)
	}

	// Limits buffer (empty block)
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write limits block: %w", err)
	}

	// SourceDefinitions count
	if err := binary.Write(&payload, binary.LittleEndian, int32(0)); err != nil {
		return nil, fmt.Errorf("write source count: %w", err)
	}

	subVersion := int16(3)
	if db != nil && db.SubVersion != 0 {
		subVersion = db.SubVersion
	}

	return encodeBlock(subVersion, payload.Bytes()), nil
}

func encodeBlock(subVersion int16, content []byte) []byte {
	var buf bytes.Buffer

	blockSize := int32(len(content) + 4 + 2 + 2)
	_ = binary.Write(&buf, binary.LittleEndian, blockSize)
	_ = binary.Write(&buf, binary.LittleEndian, int16(0))
	_ = binary.Write(&buf, binary.LittleEndian, subVersion)
	_, _ = buf.Write(content)

	return buf.Bytes()
}

func writeString(w io.Writer, value string) error {
	data := []byte(value)
	if len(data) > 0xFFFF {
		return fmt.Errorf("string too long: %d", len(data))
	}

	if err := binary.Write(w, binary.LittleEndian, uint16(len(data))); err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	_, err := w.Write(data)

	return err
}
