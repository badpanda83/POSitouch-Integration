// Package dbf provides a pure Go reader for dBASE III/IV (.dbf) files.
// No external dependencies are required.
package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// fieldDescriptor represents a single field definition in the DBF header.
type fieldDescriptor struct {
	Name         string
	Type         byte
	Length       uint8
	DecimalCount uint8
}

// ReadFile opens the DBF file at path and returns all non-deleted records as a
// slice of string maps keyed by field name.  An empty slice plus an error is
// returned when the file cannot be opened or is malformed.
func ReadFile(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return readRecords(f)
}

// readRecords does the actual parsing work against any io.ReadSeeker.
func readRecords(f io.ReadSeeker) ([]map[string]string, error) {
	// ---- header (32 bytes) -----------------------------------------------
	header := make([]byte, 32)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, fmt.Errorf("dbf: reading header: %w", err)
	}

	recordCount := binary.LittleEndian.Uint32(header[4:8])
	headerSize := binary.LittleEndian.Uint16(header[8:10])
	recordSize := binary.LittleEndian.Uint16(header[10:12])

	// ---- field descriptors (32 bytes each, terminated by 0x0D) -----------
	fieldAreaSize := int(headerSize) - 32 // bytes remaining before data
	if fieldAreaSize <= 0 {
		return nil, fmt.Errorf("dbf: invalid header size %d", headerSize)
	}

	fieldData := make([]byte, fieldAreaSize)
	if _, err := io.ReadFull(f, fieldData); err != nil {
		return nil, fmt.Errorf("dbf: reading field descriptors: %w", err)
	}

	var fields []fieldDescriptor
	for i := 0; i+32 <= len(fieldData); i += 32 {
		if fieldData[i] == 0x0D {
			break // header terminator
		}
		// Field name is null-terminated within bytes 0–10
		nameBytes := fieldData[i : i+11]
		nameEnd := 11
		for j, b := range nameBytes {
			if b == 0x00 {
				nameEnd = j
				break
			}
		}
		fields = append(fields, fieldDescriptor{
			Name:         strings.TrimSpace(string(nameBytes[:nameEnd])),
			Type:         fieldData[i+11],
			Length:       fieldData[i+16],
			DecimalCount: fieldData[i+17],
		})
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("dbf: no field descriptors found")
	}

	// ---- records ---------------------------------------------------------
	records := make([]map[string]string, 0, recordCount)
	rawRecord := make([]byte, recordSize)

	for i := uint32(0); i < recordCount; i++ {
		if _, err := io.ReadFull(f, rawRecord); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("dbf: reading record %d: %w", i, err)
		}

		// Byte 0 is the deletion flag; 0x2A ('*') means deleted.
		if rawRecord[0] == 0x2A {
			continue
		}

		rec := make(map[string]string, len(fields))
		offset := 1 // skip deletion flag
		for _, fd := range fields {
			end := offset + int(fd.Length)
			if end > len(rawRecord) {
				end = len(rawRecord)
			}
			raw := string(rawRecord[offset:end])
			rec[fd.Name] = parseField(fd, raw)
			offset = end
		}
		records = append(records, rec)
	}

	return records, nil
}

// parseField converts a raw string value from the DBF file into a clean string
// according to the field type.
func parseField(fd fieldDescriptor, raw string) string {
	switch fd.Type {
	case 'C': // Character — right-trim spaces
		return strings.TrimRight(raw, " ")
	case 'N': // Numeric — trim spaces
		return strings.TrimSpace(raw)
	case 'D': // Date YYYYMMDD — return as-is (trimmed)
		return strings.TrimSpace(raw)
	case 'L': // Logical T/F/Y/N
		switch strings.ToUpper(strings.TrimSpace(raw)) {
		case "T", "Y":
			return "true"
		case "F", "N":
			return "false"
		default:
			return "false"
		}
	default:
		return strings.TrimRight(raw, " ")
	}
}
