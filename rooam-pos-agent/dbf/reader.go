// Package dbf provides a pure Go reader for dBASE III/IV (.dbf) files.
// No external dependencies are required.
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

// fieldDescriptor holds metadata for a single DBF field.
type fieldDescriptor struct {
	Name     string
	Type     byte
	Size     uint8
	Decimals uint8
}

// ReadFile opens a DBF file at the given path and returns all non-deleted
// records as a slice of maps keyed by field name.
func ReadFile(path string) ([]map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dbf: open %q: %w", path, err)
	}
	defer f.Close()
	return Read(f)
}

// Read reads DBF data from the given io.ReadSeeker and returns all
// non-deleted records.
func Read(r io.ReadSeeker) ([]map[string]interface{}, error) {
	// --- Header ---
	// Bytes 0–3: version(1), YYMMDD last update(3)
	// Bytes 4–7: number of records (uint32 LE)
	// Bytes 8–9: header size (uint16 LE)
	// Bytes 10–11: record size (uint16 LE)
	header := make([]byte, 32)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("dbf: read header: %w", err)
	}

	numRecords := binary.LittleEndian.Uint32(header[4:8])
	headerSize := binary.LittleEndian.Uint16(header[8:10])
	recordSize := binary.LittleEndian.Uint16(header[10:12])

	// --- Field descriptors ---
	// Each field descriptor is 32 bytes; terminated by 0x0D.
	// Number of fields = (headerSize - 32 - 1) / 32
	numFields := (int(headerSize) - 32 - 1) / 32
	if numFields <= 0 {
		return nil, fmt.Errorf("dbf: invalid header size %d", headerSize)
	}

	fields := make([]fieldDescriptor, 0, numFields)
	for i := 0; i < numFields; i++ {
		fd := make([]byte, 32)
		if _, err := io.ReadFull(r, fd); err != nil {
			return nil, fmt.Errorf("dbf: read field descriptor %d: %w", i, err)
		}
		// Field name is in bytes 0–10, null-terminated
		nameBytes := fd[0:11]
		nameEnd := 11
		for j := 0; j < 11; j++ {
			if nameBytes[j] == 0 {
				nameEnd = j
				break
			}
		}
		fields = append(fields, fieldDescriptor{
			Name:     strings.TrimRight(string(nameBytes[:nameEnd]), " \x00"),
			Type:     fd[11],
			Size:     fd[16],
			Decimals: fd[17],
		})
	}

	// Seek to start of records (skip the 0x0D terminator and any padding)
	if _, err := r.Seek(int64(headerSize), io.SeekStart); err != nil {
		return nil, fmt.Errorf("dbf: seek to records: %w", err)
	}

	// --- Records ---
	records := make([]map[string]interface{}, 0, numRecords)
	buf := make([]byte, recordSize)
	for i := uint32(0); i < numRecords; i++ {
		if _, err := io.ReadFull(r, buf); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("dbf: read record %d: %w", i, err)
		}

		// First byte is the deletion flag: 0x2A ('*') means deleted
		if buf[0] == 0x2A {
			continue
		}

		rec := make(map[string]interface{}, len(fields))
		offset := 1 // skip deletion flag
		for _, fd := range fields {
			raw := string(buf[offset : offset+int(fd.Size)])
			offset += int(fd.Size)
			rec[fd.Name] = parseField(fd, raw)
		}
		records = append(records, rec)
	}

	return records, nil
}

// parseField converts the raw string value of a DBF field into a Go value.
func parseField(fd fieldDescriptor, raw string) interface{} {
	switch fd.Type {
	case 'C': // Character
		return strings.TrimRight(raw, " ")
	case 'N': // Numeric
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || trimmed == "." {
			return float64(0)
		}
		v, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return float64(0)
		}
		// If no decimal places, return as integer-compatible float
		if fd.Decimals == 0 {
			return math.Trunc(v)
		}
		return v
	case 'D': // Date (YYYYMMDD)
		trimmed := strings.TrimSpace(raw)
		if len(trimmed) == 8 {
			// Return as ISO-style string YYYY-MM-DD
			return trimmed[:4] + "-" + trimmed[4:6] + "-" + trimmed[6:8]
		}
		return trimmed
	case 'L': // Logical — T/Y/1 are truthy; case-insensitive via ToUpper; '1' is digit, unaffected by ToUpper
		switch strings.ToUpper(strings.TrimSpace(raw)) {
		case "T", "Y", "1":
			return true
		default:
			return false
		}
	default:
		return strings.TrimRight(raw, " ")
	}
}
