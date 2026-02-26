// Package dbf provides a pure Go reader for dBASE III/IV .dbf files.
// No external dependencies — uses only the Go standard library.
package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Field describes a single column in a DBF file.
type Field struct {
	Name    string
	Type    byte
	Length  int
	Decimal int
}

// Record is a single row of data, keyed by field name.
type Record map[string]interface{}

// DBFFile holds the parsed fields and records from a .dbf file.
type DBFFile struct {
	Fields  []Field
	Records []Record
}

// Open reads the DBF file at path and returns all non-deleted records.
// Supports field types C (character), N (numeric), D (date), and L (logical).
func Open(path string) (*DBFFile, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("dbf: open %s: %w", path, err)
	}
	defer f.Close()

	// --- Read header (32 bytes) ---
	header := make([]byte, 32)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, fmt.Errorf("dbf: read header: %w", err)
	}

	recordCount := binary.LittleEndian.Uint32(header[4:8])
	headerSize := binary.LittleEndian.Uint16(header[8:10])
	recordSize := binary.LittleEndian.Uint16(header[10:12])

	// --- Read field descriptors (32 bytes each, terminated by 0x0D) ---
	fieldAreaSize := int(headerSize) - 32
	if fieldAreaSize <= 0 {
		return nil, fmt.Errorf("dbf: invalid header size %d", headerSize)
	}
	fieldData := make([]byte, fieldAreaSize)
	if _, err := io.ReadFull(f, fieldData); err != nil {
		return nil, fmt.Errorf("dbf: read field descriptors: %w", err)
	}

	var fields []Field
	for i := 0; i+32 <= len(fieldData); i += 32 {
		if fieldData[i] == 0x0D {
			break
		}
		nameBytes := fieldData[i : i+11]
		// Null-terminate the name
		n := 0
		for n < 11 && nameBytes[n] != 0x00 {
			n++
		}
		name := string(nameBytes[:n])
		fieldType := fieldData[i+11]
		length := int(fieldData[i+16])
		decimal := int(fieldData[i+17])

		fields = append(fields, Field{
			Name:    name,
			Type:    fieldType,
			Length:  length,
			Decimal: decimal,
		})
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("dbf: no fields found in %s", path)
	}

	// --- Read records ---
	var records []Record
	recBuf := make([]byte, int(recordSize))
	for i := uint32(0); i < recordCount; i++ {
		if _, err := io.ReadFull(f, recBuf); err != nil {
			// Truncated file — stop reading
			break
		}
		// First byte is the deletion flag
		if recBuf[0] == 0x2A { // '*' = deleted
			continue
		}

		rec := make(Record, len(fields))
		offset := 1 // skip deletion flag
		for _, fld := range fields {
			raw := string(recBuf[offset : offset+fld.Length])
			offset += fld.Length
			rec[fld.Name] = parseField(fld, raw)
		}
		records = append(records, rec)
	}

	return &DBFFile{Fields: fields, Records: records}, nil
}

func parseField(fld Field, raw string) interface{} {
	switch fld.Type {
	case 'C':
		return strings.TrimRight(raw, " ")
	case 'N':
		s := strings.TrimSpace(raw)
		if s == "" {
			return float64(0)
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return float64(0)
		}
		return v
	case 'D':
		// YYYYMMDD — return as-is (already a string)
		return strings.TrimSpace(raw)
	case 'L':
		if len(raw) == 0 {
			return false
		}
		switch raw[0] {
		case 'Y', 'y', 'T', 't':
			return true
		default:
			return false
		}
	default:
		return strings.TrimRight(raw, " ")
	}
}
