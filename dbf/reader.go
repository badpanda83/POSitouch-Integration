// Package dbf provides a minimal dBASE III/IV (.dbf) file reader.
// It supports field types C (character), N (numeric), D (date), and L (logical).
package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	headerSize    = 32
	fieldDescSize = 32
	headerTerminator = 0x0D
	recordDeleted    = 0x2A
	recordActive     = 0x20
)

// FieldDescriptor describes a single field in a DBF file.
type FieldDescriptor struct {
	Name     string
	Type     byte   // 'C', 'N', 'D', 'L'
	Length   int
	Decimals int
	Offset   int    // byte offset within each record (computed by reader)
}

// Record is a map from field name to the parsed value.
type Record map[string]interface{}

// Reader reads records from a dBASE III/IV (.dbf) file.
type Reader struct {
	Fields  []FieldDescriptor
	records []Record
}

// Open opens the named DBF file and reads all non-deleted records into memory.
func Open(filename string) (*Reader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("dbf.Open %s: %w", filename, err)
	}
	defer f.Close()
	return read(f)
}

func read(r io.Reader) (*Reader, error) {
	// --- header ---
	hdr := make([]byte, headerSize)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	numRecords := int(binary.LittleEndian.Uint32(hdr[4:8]))
	headerBytes := int(binary.LittleEndian.Uint16(hdr[8:10]))
	recordBytes := int(binary.LittleEndian.Uint16(hdr[10:12]))

	// --- field descriptors ---
	numFields := (headerBytes - headerSize - 1) / fieldDescSize
	fields := make([]FieldDescriptor, 0, numFields)
	offset := 1 // first byte of each record is the deletion flag

	for i := 0; i < numFields; i++ {
		fdBuf := make([]byte, fieldDescSize)
		if _, err := io.ReadFull(r, fdBuf); err != nil {
			return nil, fmt.Errorf("reading field descriptor %d: %w", i, err)
		}
		// A 0x0D byte marks the end of field descriptors.
		if fdBuf[0] == headerTerminator {
			break
		}

		nameBytes := fdBuf[0:11]
		// Trim null bytes from field name.
		nameLen := 0
		for nameLen < len(nameBytes) && nameBytes[nameLen] != 0 {
			nameLen++
		}
		name := strings.TrimSpace(string(nameBytes[:nameLen]))

		fd := FieldDescriptor{
			Name:     strings.ToUpper(name),
			Type:     fdBuf[11],
			Length:   int(fdBuf[16]),
			Decimals: int(fdBuf[17]),
			Offset:   offset,
		}
		fields = append(fields, fd)
		offset += fd.Length
	}

	// Consume the rest of the header (terminator + possible header filler).
	headerRead := headerSize + len(fields)*fieldDescSize
	remaining := headerBytes - headerRead
	if remaining > 0 {
		discard := make([]byte, remaining)
		if _, err := io.ReadFull(r, discard); err != nil {
			return nil, fmt.Errorf("discarding header remainder: %w", err)
		}
	}

	// --- records ---
	records := make([]Record, 0, numRecords)
	recBuf := make([]byte, recordBytes)

	for i := 0; i < numRecords; i++ {
		n, err := io.ReadFull(r, recBuf)
		if err != nil || n == 0 {
			break
		}
		// Skip deleted records.
		if recBuf[0] == recordDeleted {
			continue
		}

		rec := make(Record, len(fields))
		for _, fd := range fields {
			raw := string(recBuf[fd.Offset : fd.Offset+fd.Length])
			rec[fd.Name] = parseField(fd, raw)
		}
		records = append(records, rec)
	}

	return &Reader{Fields: fields, records: records}, nil
}

// Records returns all non-deleted records read from the file.
func (r *Reader) Records() []Record {
	return r.records
}

// parseField converts the raw string representation of a field into a Go value.
func parseField(fd FieldDescriptor, raw string) interface{} {
	switch fd.Type {
	case 'C':
		return strings.TrimRight(raw, " ")
	case 'N':
		s := strings.TrimSpace(raw)
		if s == "" {
			return 0.0
		}
		if fd.Decimals > 0 {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return 0.0
			}
			return v
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			// Try float fallback for malformed integer fields.
			f, err2 := strconv.ParseFloat(s, 64)
			if err2 != nil {
				return int64(0)
			}
			return int64(f)
		}
		return v
	case 'D':
		return strings.TrimSpace(raw) // YYYYMMDD string; callers can parse further if needed
	case 'L':
		s := strings.ToUpper(strings.TrimSpace(raw))
		return s == "T" || s == "Y"
	default:
		return strings.TrimRight(raw, " ")
	}
}

// GetString returns the value of a field as a string, or "" if absent/wrong type.
func (r Record) GetString(field string) string {
	v, ok := r[field]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

// GetInt returns the value of a field as int64, or 0 if absent/wrong type.
func (r Record) GetInt(field string) int64 {
	v, ok := r[field]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case int64:
		return t
	case float64:
		return int64(t)
	}
	return 0
}

// GetFloat returns the value of a field as float64, or 0 if absent/wrong type.
func (r Record) GetFloat(field string) float64 {
	v, ok := r[field]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int64:
		return float64(t)
	}
	return 0
}

// GetBool returns the value of a logical field as bool, or false if absent/wrong type.
func (r Record) GetBool(field string) bool {
	v, ok := r[field]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}
