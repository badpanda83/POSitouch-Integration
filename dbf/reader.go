// Package dbf provides a minimal reader for dBASE III/IV (.dbf) files.
//
// The implementation follows the well-documented dBASE III Plus file format:
//   - 32-byte file header
//   - 32-byte field descriptor records, terminated by 0x0D
//   - Fixed-length data records
//
// Supported field types: C (Character), N (Numeric), D (Date), L (Logical).
package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// FieldDescriptor describes one column in a DBF table.
type FieldDescriptor struct {
	Name      string
	Type      byte   // 'C', 'N', 'D', 'L'
	Length    uint8  // field length in bytes
	Decimals  uint8  // decimal count for N fields
	Offset    int    // byte offset within the data record (computed)
}

// Table holds the parsed contents of a DBF file.
type Table struct {
	Fields  []FieldDescriptor
	Records []Record
}

// Record is a single row expressed as a map of field-name → value.
// Values are always Go strings; callers convert as needed.
type Record map[string]string

// Open reads the DBF file at path and returns the parsed Table.
func Open(path string) (*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dbf: open %s: %w", path, err)
	}
	defer f.Close()
	return Read(f)
}

// Read parses a DBF file from an io.Reader.
func Read(r io.Reader) (*Table, error) {
	// ── File header (32 bytes) ──────────────────────────────────────────────
	header := make([]byte, 32)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("dbf: read header: %w", err)
	}

	numRecords := binary.LittleEndian.Uint32(header[4:8])
	headerBytes := binary.LittleEndian.Uint16(header[8:10])
	recordSize := binary.LittleEndian.Uint16(header[10:12])

	// ── Field descriptors ───────────────────────────────────────────────────
	// Read the remainder of the header block into memory so we can parse field
	// descriptors without risking over-reads.  headerBytes includes the 32-byte
	// file header, so remaining = headerBytes - 32.
	rest := make([]byte, int(headerBytes)-32)
	if _, err := io.ReadFull(r, rest); err != nil {
		return nil, fmt.Errorf("dbf: read field descriptors: %w", err)
	}

	fields := make([]FieldDescriptor, 0, len(rest)/32)
	offset := 1 // first byte of each record is the deletion flag

	for pos := 0; pos+32 <= len(rest); pos += 32 {
		if rest[pos] == 0x0D {
			break // header terminator
		}
		desc := rest[pos : pos+32]

		// Field name is null-padded in bytes 0-10.
		nameBytes := desc[0:11]
		end := 0
		for end < len(nameBytes) && nameBytes[end] != 0 {
			end++
		}
		name := strings.TrimSpace(string(nameBytes[:end]))

		fd := FieldDescriptor{
			Name:     strings.ToUpper(name),
			Type:     desc[11],
			Length:   desc[16],
			Decimals: desc[17],
			Offset:   offset,
		}
		fields = append(fields, fd)
		offset += int(fd.Length)
	}

	// ── Data records ────────────────────────────────────────────────────────
	records := make([]Record, 0, numRecords)
	rowBuf := make([]byte, int(recordSize))

	for i := uint32(0); i < numRecords; i++ {
		if _, err := io.ReadFull(r, rowBuf); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("dbf: read record %d: %w", i, err)
		}

		// Byte 0 is the deletion flag: ' ' = active, '*' = deleted.
		if rowBuf[0] == '*' {
			continue
		}

		rec := make(Record, len(fields))
		for _, fd := range fields {
			raw := string(rowBuf[fd.Offset : fd.Offset+int(fd.Length)])
			rec[fd.Name] = parseField(fd.Type, raw)
		}
		records = append(records, rec)
	}

	return &Table{Fields: fields, Records: records}, nil
}

// parseField converts the raw bytes of a field to a canonical string value.
func parseField(fieldType byte, raw string) string {
	switch fieldType {
	case 'C':
		return strings.TrimRight(raw, " ")
	case 'N':
		return strings.TrimSpace(raw)
	case 'D':
		// dBASE date: YYYYMMDD → keep as-is
		return strings.TrimSpace(raw)
	case 'L':
		switch raw[0] {
		case 'T', 't', 'Y', 'y':
			return "true"
		default:
			return "false"
		}
	default:
		return strings.TrimSpace(raw)
	}
}

// FieldInt reads a named field from a record as an integer.
// Returns 0 and false if the field is absent or cannot be parsed.
func FieldInt(rec Record, name string) (int, bool) {
	v, ok := rec[name]
	if !ok || v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, false
	}
	return n, true
}

// FieldString reads a named field from a record as a trimmed string.
func FieldString(rec Record, name string) string {
	return strings.TrimSpace(rec[name])
}
