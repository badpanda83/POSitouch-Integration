// Package dbf provides a pure Go reader for dBASE III/IV (.DBF) files.
// No CGo, no external dependencies.
package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

// Field describes a single column in a DBF file.
type Field struct {
	Name     string
	Type     byte   // 'C', 'N', 'D', 'L'
	Length   int
	Decimals int
}

// Record is a map of field name → parsed value.
// Values are string, float64, bool, or nil (for deleted/blank).
type Record map[string]interface{}

// File holds the parsed header and provides row iteration.
type File struct {
	Fields  []Field
	records []Record
}

// Open reads and fully parses the DBF file at path.
func Open(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parse(f)
}

// Records returns all non-deleted records.
func (f *File) Records() []Record {
	return f.records
}

// FieldNames returns the ordered list of field names.
func (f *File) FieldNames() []string {
	names := make([]string, len(f.Fields))
	for i, fld := range f.Fields {
		names[i] = fld.Name
	}
	return names
}

// --- internal parsing ---

func parse(r io.ReadSeeker) (*File, error) {
	var hdr [32]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("dbf: reading header: %w", err)
	}

	recordCnt := binary.LittleEndian.Uint32(hdr[4:8])
	headerLen := binary.LittleEndian.Uint16(hdr[8:10])
	recordLen := binary.LittleEndian.Uint16(hdr[10:12])

	// Field descriptors start at offset 32 and are 32 bytes each.
	// They end at headerLen-1 (the last byte is the terminator 0x0D).
	numFields := (int(headerLen) - 32 - 1) / 32
	if numFields < 1 {
		return nil, fmt.Errorf("dbf: invalid header: numFields=%d", numFields)
	}

	fields := make([]Field, 0, numFields)
	for i := 0; i < numFields; i++ {
		var fd [32]byte
		if _, err := io.ReadFull(r, fd[:]); err != nil {
			return nil, fmt.Errorf("dbf: reading field descriptor %d: %w", i, err)
		}
		// Null-terminated name in first 11 bytes
		nameBytes := fd[0:11]
		end := 11
		for j := 0; j < 11; j++ {
			if nameBytes[j] == 0 {
				end = j
				break
			}
		}
		fields = append(fields, Field{
			Name:     strings.TrimRight(string(nameBytes[:end]), "\x00 "),
			Type:     fd[11],
			Length:   int(fd[16]),
			Decimals: int(fd[17]),
		})
	}

	// Seek to the start of data records (headerLen accounts for the terminator)
	if _, err := r.Seek(int64(headerLen), io.SeekStart); err != nil {
		return nil, fmt.Errorf("dbf: seeking to records: %w", err)
	}

	records := make([]Record, 0, recordCnt)
	recBuf := make([]byte, recordLen)
	for i := uint32(0); i < recordCnt; i++ {
		n, err := io.ReadFull(r, recBuf)
		if err != nil || n != int(recordLen) {
			break // truncated file — stop reading
		}
		// First byte: 0x20 = active, 0x2A = deleted
		if recBuf[0] == 0x2A {
			continue
		}
		rec := make(Record, len(fields))
		offset := 1 // skip deletion flag
		for _, fld := range fields {
			raw := string(recBuf[offset : offset+fld.Length])
			offset += fld.Length
			rec[fld.Name] = parseField(fld.Type, raw, fld.Decimals)
		}
		records = append(records, rec)
	}

	return &File{Fields: fields, records: records}, nil
}

func parseField(typ byte, raw string, decimals int) interface{} {
	switch typ {
	case 'C': // Character — trim trailing spaces
		return strings.TrimRight(raw, " ")
	case 'N': // Numeric
		s := strings.TrimSpace(raw)
		if s == "" || s == "." {
			return float64(0)
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return float64(0)
		}
		if decimals == 0 && !math.IsInf(v, 0) && !math.IsNaN(v) {
			return math.Round(v)
		}
		return v
	case 'D': // Date YYYYMMDD
		return strings.TrimSpace(raw)
	case 'L': // Logical
		if len(raw) == 0 {
			return false
		}
		c := raw[0]
		return c == 'T' || c == 't' || c == 'Y' || c == 'y'
	default:
		return strings.TrimRight(raw, " ")
	}
}

// GetString is a helper that returns a field value as string.
func GetString(rec Record, field string) string {
	v, ok := rec[field]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// GetFloat64 is a helper that returns a field value as float64.
func GetFloat64(rec Record, field string) float64 {
	v, ok := rec[field]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	}
	return 0
}

// GetInt is a helper that returns a field value as int.
func GetInt(rec Record, field string) int {
	return int(GetFloat64(rec, field))
}
