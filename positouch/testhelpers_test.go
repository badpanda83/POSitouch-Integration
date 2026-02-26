package positouch

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// fieldSpec describes a single DBF field to be included in a test file.
type fieldSpec struct {
	name     string
	typ      byte
	size     uint8
	decimals uint8
}

// buildDBF creates a minimal dBASE III DBF file in memory with the given
// fields and records. Each record is a slice of raw string values in field order.
func buildDBF(fields []fieldSpec, rows [][]string) []byte {
	numFields := len(fields)
	headerSize := uint16(32 + numFields*32 + 1)

	recordSize := uint16(1) // deletion flag
	for _, f := range fields {
		recordSize += uint16(f.size)
	}

	buf := new(bytes.Buffer)

	// File header (32 bytes)
	header := make([]byte, 32)
	header[0] = 0x03
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(rows)))
	binary.LittleEndian.PutUint16(header[8:10], headerSize)
	binary.LittleEndian.PutUint16(header[10:12], recordSize)
	buf.Write(header)

	// Field descriptors (32 bytes each)
	for _, f := range fields {
		fd := make([]byte, 32)
		copy(fd[0:11], f.name)
		fd[11] = f.typ
		fd[16] = f.size
		fd[17] = f.decimals
		buf.Write(fd)
	}

	// Header terminator
	buf.WriteByte(0x0D)

	// Pad to exact headerSize
	for buf.Len() < int(headerSize) {
		buf.WriteByte(0x00)
	}

	// Records
	for _, row := range rows {
		buf.WriteByte(0x20) // active record
		for i, f := range fields {
			val := make([]byte, f.size)
			if i < len(row) {
				copy(val, row[i])
			}
			buf.Write(val)
		}
	}

	return buf.Bytes()
}

// writeTempDBF writes a DBF byte slice to a temp directory and returns the path.
func writeTempDBF(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatalf("writeTempDBF %s: %v", name, err)
	}
	return p
}
