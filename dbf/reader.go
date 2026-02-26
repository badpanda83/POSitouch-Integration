// Package dbf provides a pure Go dBASE III/IV file reader.
package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// FieldDescriptor holds metadata for a single DBF field.
type FieldDescriptor struct {
	Name        string
	Type        byte
	Length      uint8
	DecimalCount uint8
}

// ReadFile opens a DBF file and returns all non-deleted records as a slice of maps.
// Supported field types: C (Character), N (Numeric), D (Date), L (Logical).
func ReadFile(path string) ([]map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dbf: open %q: %w", path, err)
	}
	defer f.Close()

	return readFrom(f)
}

func readFrom(r io.ReadSeeker) ([]map[string]interface{}, error) {
	// --- Header ---
	// Byte 0: version
	// Bytes 1-3: last update (YY MM DD)
	// Bytes 4-7: number of records (little-endian uint32)
	// Bytes 8-9: header size in bytes (little-endian uint16)
	// Bytes 10-11: record size in bytes (little-endian uint16)
	// Bytes 12-31: reserved

	header := make([]byte, 32)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("dbf: read header: %w", err)
	}

	recordCount := binary.LittleEndian.Uint32(header[4:8])
	headerSize := binary.LittleEndian.Uint16(header[8:10])
	recordSize := binary.LittleEndian.Uint16(header[10:12])

	// --- Field descriptors ---
	// Each field descriptor is 32 bytes; they end with a 0x0D terminator.
	numFields := (int(headerSize) - 32 - 1) / 32
	if numFields <= 0 {
		return nil, fmt.Errorf("dbf: invalid header size %d", headerSize)
	}

	fields := make([]FieldDescriptor, 0, numFields)
	for i := 0; i < numFields; i++ {
		fdBuf := make([]byte, 32)
		if _, err := io.ReadFull(r, fdBuf); err != nil {
			return nil, fmt.Errorf("dbf: read field descriptor %d: %w", i, err)
		}
		// Terminator byte — end of field descriptors
		if fdBuf[0] == 0x0D {
			break
		}
		nameBytes := fdBuf[0:11]
		name := strings.ToUpper(strings.TrimRight(string(nameBytes), "\x00"))
		fields = append(fields, FieldDescriptor{
			Name:        name,
			Type:        fdBuf[11],
			Length:      fdBuf[16],
			DecimalCount: fdBuf[17],
		})
	}

	// Seek to start of records
	if _, err := r.Seek(int64(headerSize), io.SeekStart); err != nil {
		return nil, fmt.Errorf("dbf: seek to records: %w", err)
	}

	// --- Records ---
	records := make([]map[string]interface{}, 0, recordCount)
	recBuf := make([]byte, recordSize)

	for i := uint32(0); i < recordCount; i++ {
		if _, err := io.ReadFull(r, recBuf); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, fmt.Errorf("dbf: read record %d: %w", i, err)
		}

		// Byte 0 of record: 0x20 = active, 0x2A (*) = deleted
		if recBuf[0] == 0x2A {
			continue
		}

		rec := make(map[string]interface{}, len(fields))
		offset := 1 // skip deletion flag byte

		for _, fd := range fields {
			raw := string(recBuf[offset : offset+int(fd.Length)])
			offset += int(fd.Length)

			switch fd.Type {
			case 'C':
				rec[fd.Name] = strings.TrimSpace(raw)
			case 'N':
				s := strings.TrimSpace(raw)
				if s == "" {
					rec[fd.Name] = float64(0)
				} else {
					v, err := strconv.ParseFloat(s, 64)
					if err != nil {
						rec[fd.Name] = float64(0)
					} else {
						rec[fd.Name] = v
					}
				}
			case 'D':
				// YYYYMMDD
				s := strings.TrimSpace(raw)
				if len(s) == 8 {
					rec[fd.Name] = s[0:4] + "-" + s[4:6] + "-" + s[6:8]
				} else {
					rec[fd.Name] = ""
				}
			case 'L':
				s := strings.ToUpper(strings.TrimSpace(raw))
				rec[fd.Name] = s == "T" || s == "Y"
			default:
				rec[fd.Name] = strings.TrimSpace(raw)
			}
		}

		records = append(records, rec)
	}

	return records, nil
}
