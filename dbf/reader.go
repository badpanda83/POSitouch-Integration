// Package dbf provides a pure Go reader for dBASE III/IV (.DBF) files.
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
	Length   uint8
	Decimals uint8
}

// ReadFile opens and reads all active (non-deleted) records from a DBF file.
// It returns a slice of maps where each key is the field name and each value
// is the parsed Go value: string (C), float64 (N), string (D), bool (L).
func ReadFile(path string) ([]map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return readFromReader(f)
}

func readFromReader(r io.ReadSeeker) ([]map[string]interface{}, error) {
	// ── File header (32 bytes) ──────────────────────────────────────────────
	header := make([]byte, 32)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("dbf: reading header: %w", err)
	}

	numRecords := binary.LittleEndian.Uint32(header[4:8])
	headerSize := binary.LittleEndian.Uint16(header[8:10])
	recordSize := binary.LittleEndian.Uint16(header[10:12])

	// ── Field descriptors ──────────────────────────────────────────────────
	// Each descriptor is 32 bytes; the list ends with a 0x0D terminator byte.
	// Number of descriptors = (headerSize - 32 - 1) / 32
	numFields := (int(headerSize) - 32 - 1) / 32
	if numFields <= 0 {
		return nil, fmt.Errorf("dbf: invalid header size %d", headerSize)
	}

	fields := make([]fieldDescriptor, 0, numFields)
	for i := 0; i < numFields; i++ {
		desc := make([]byte, 32)
		if _, err := io.ReadFull(r, desc); err != nil {
			return nil, fmt.Errorf("dbf: reading field descriptor %d: %w", i, err)
		}
		// A 0x0D byte in position 0 means the header terminator was reached early.
		if desc[0] == 0x0D {
			break
		}
		// Field name is null-terminated within the first 11 bytes.
		nameBytes := desc[0:11]
		nameEnd := 11
		for j := 0; j < 11; j++ {
			if nameBytes[j] == 0 {
				nameEnd = j
				break
			}
		}
		fields = append(fields, fieldDescriptor{
			Name:     strings.TrimRight(string(nameBytes[:nameEnd]), " \x00"),
			Type:     desc[11],
			Length:   desc[16],
			Decimals: desc[17],
		})
	}

	// Seek to the start of the record area (absolute position = headerSize).
	if _, err := r.Seek(int64(headerSize), io.SeekStart); err != nil {
		return nil, fmt.Errorf("dbf: seeking to records: %w", err)
	}

	// ── Records ────────────────────────────────────────────────────────────
	records := make([]map[string]interface{}, 0, numRecords)
	recBuf := make([]byte, recordSize)

	for i := uint32(0); i < numRecords; i++ {
		if _, err := io.ReadFull(r, recBuf); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("dbf: reading record %d: %w", i, err)
		}

		// 0x2A ('*') marks a deleted record; 0x20 (' ') marks an active one.
		if recBuf[0] == 0x2A {
			continue
		}

		row := make(map[string]interface{}, len(fields))
		offset := 1 // skip deletion flag byte
		for _, fd := range fields {
			raw := string(recBuf[offset : offset+int(fd.Length)])
			offset += int(fd.Length)
			row[fd.Name] = parseField(fd, raw)
		}
		records = append(records, row)
	}

	return records, nil
}

// parseField converts raw DBF field bytes to an appropriate Go type.
func parseField(fd fieldDescriptor, raw string) interface{} {
	switch fd.Type {
	case 'C':
		return strings.TrimRight(raw, " ")
	case 'N':
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || trimmed == "." {
			return float64(0)
		}
		v, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return float64(0)
		}
		if fd.Decimals == 0 {
			return math.Trunc(v)
		}
		return v
	case 'D':
		trimmed := strings.TrimSpace(raw)
		if len(trimmed) == 8 {
			// Return as ISO-style string YYYY-MM-DD
			return trimmed[:4] + "-" + trimmed[4:6] + "-" + trimmed[6:8]
		}
		return trimmed
	case 'L':
		// Standard DBF values are T/Y (true) and F/N (false).
		// '1' is accepted as truthy to accommodate non-standard POSitouch exports.
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
