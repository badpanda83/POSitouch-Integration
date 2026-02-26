// Package dbf provides a pure Go reader for dBASE III/IV (.dbf) files.
// No CGO or external dependencies are used.
package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// fieldDescriptor describes a single field in the DBF header.
type fieldDescriptor struct {
	Name     string
	Type     byte
	Length   uint8
	Decimals uint8
}

// ReadFile opens a DBF file and returns all non-deleted records as a slice of
// map[string]interface{} where keys are uppercased field names.
func ReadFile(path string) ([]map[string]interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Read(f)
}

// Read parses DBF data from any io.ReadSeeker.
func Read(r io.ReadSeeker) ([]map[string]interface{}, error) {
	// --- Header (first 32 bytes) ---
	var hdr [32]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("dbf: reading header: %w", err)
	}

	numRecords := binary.LittleEndian.Uint32(hdr[4:8])
	headerBytes := binary.LittleEndian.Uint16(hdr[8:10])
	recordSize := binary.LittleEndian.Uint16(hdr[10:12])

	// --- Field descriptors (32 bytes each, terminated by 0x0D) ---
	// Field descriptors start at byte 32 and end at headerBytes-1.
	numFields := (int(headerBytes) - 32 - 1) / 32
	if numFields <= 0 {
		return nil, fmt.Errorf("dbf: invalid header: headerBytes=%d", headerBytes)
	}

	fields := make([]fieldDescriptor, 0, numFields)
	for i := 0; i < numFields; i++ {
		var fd [32]byte
		if _, err := io.ReadFull(r, fd[:]); err != nil {
			return nil, fmt.Errorf("dbf: reading field descriptor %d: %w", i, err)
		}
		if fd[0] == 0x0D {
			break
		}
		nameBytes := fd[0:11]
		name := strings.TrimRight(string(nameBytes), "\x00")
		fields = append(fields, fieldDescriptor{
			Name:     strings.ToUpper(name),
			Type:     fd[11],
			Length:   fd[16],
			Decimals: fd[17],
		})
	}

	// Seek to the start of records.
	if _, err := r.Seek(int64(headerBytes), io.SeekStart); err != nil {
		return nil, fmt.Errorf("dbf: seeking to records: %w", err)
	}

	records := make([]map[string]interface{}, 0, numRecords)
	recBuf := make([]byte, recordSize)

	for i := uint32(0); i < numRecords; i++ {
		if _, err := io.ReadFull(r, recBuf); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			log.Printf("dbf: warning: reading record %d: %v", i, err)
			continue
		}

		// Skip deleted records (deletion flag byte == '*').
		if recBuf[0] == '*' {
			continue
		}

		rec := make(map[string]interface{}, len(fields))
		offset := 1 // skip deletion flag byte
		for _, fd := range fields {
			end := offset + int(fd.Length)
			if end > len(recBuf) {
				log.Printf("dbf: warning: record %d field %s extends beyond record boundary", i, fd.Name)
				offset = end
				continue
			}
			raw := string(recBuf[offset:end])
			rec[fd.Name] = parseField(fd, raw, i)
			offset = end
		}
		records = append(records, rec)
	}

	return records, nil
}

// parseField converts a raw string value to the appropriate Go type based on
// the field descriptor.
func parseField(fd fieldDescriptor, raw string, recIdx uint32) interface{} {
	switch fd.Type {
	case 'C':
		return strings.TrimRight(raw, " ")

	case 'N':
		s := strings.TrimSpace(raw)
		if s == "" || s == "." {
			return nil
		}
		if fd.Decimals == 0 {
			v, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				log.Printf("dbf: warning: record %d field %s: cannot parse int %q: %v", recIdx, fd.Name, s, err)
				return nil
			}
			return int(v)
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			log.Printf("dbf: warning: record %d field %s: cannot parse float %q: %v", recIdx, fd.Name, s, err)
			return nil
		}
		// Round to the declared decimal places to avoid floating-point noise.
		factor := math.Pow(10, float64(fd.Decimals))
		return math.Round(v*factor) / factor

	case 'D':
		s := strings.TrimSpace(raw)
		if len(s) != 8 || s == "00000000" {
			return nil
		}
		t, err := time.Parse("20060102", s)
		if err != nil {
			log.Printf("dbf: warning: record %d field %s: cannot parse date %q: %v", recIdx, fd.Name, s, err)
			return nil
		}
		return t

	case 'L':
		switch strings.ToUpper(strings.TrimSpace(raw)) {
		case "T", "Y":
			return true
		case "F", "N":
			return false
		default:
			return nil
		}

	default:
		return strings.TrimRight(raw, " ")
	}
}
