package dbf

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// buildDBF constructs a minimal valid dBASE III DBF byte stream with a single
// Character field and one record.
func buildDBF(t *testing.T) []byte {
	t.Helper()

	// One field descriptor: "NAME", type C, size 10
	fieldName := [11]byte{}
	copy(fieldName[:], "NAME")
	fieldType := byte('C')
	fieldSize := uint8(10)

	// Header is 32 bytes; one field is 32 bytes; terminator is 1 byte → 65 total
	headerSize := uint16(32 + 32 + 1) // 65
	recordSize := uint16(1 + 10)      // deletion flag + field

	var buf bytes.Buffer

	// --- DBF header (32 bytes) ---
	header := make([]byte, 32)
	header[0] = 0x03                                        // dBASE III
	binary.LittleEndian.PutUint32(header[4:8], 1)           // 1 record
	binary.LittleEndian.PutUint16(header[8:10], headerSize) // header size
	binary.LittleEndian.PutUint16(header[10:12], recordSize)
	buf.Write(header)

	// --- Field descriptor (32 bytes) ---
	fd := make([]byte, 32)
	copy(fd[0:11], fieldName[:])
	fd[11] = fieldType
	fd[16] = fieldSize
	buf.Write(fd)

	// --- Terminator ---
	buf.WriteByte(0x0D)

	// --- Record: not deleted, value "Hello     " ---
	buf.WriteByte(0x20) // not deleted
	rec := make([]byte, 10)
	copy(rec, "Hello     ")
	buf.Write(rec)

	return buf.Bytes()
}

func TestRead_SingleCharacterField(t *testing.T) {
	data := buildDBF(t)
	r := bytes.NewReader(data)

	records, err := Read(r)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	got, ok := records[0]["NAME"].(string)
	if !ok {
		t.Fatalf("NAME field is not string, got %T", records[0]["NAME"])
	}
	if got != "Hello" {
		t.Errorf("NAME = %q, want %q", got, "Hello")
	}
}

func TestRead_DeletedRecordSkipped(t *testing.T) {
	data := buildDBF(t)
	// Mark the record as deleted by setting byte 65 (first byte of record) to '*'
	data[65] = 0x2A // '*'

	r := bytes.NewReader(data)
	records, err := Read(r)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records (deleted), got %d", len(records))
	}
}

func TestParseField_Numeric(t *testing.T) {
	fd := fieldDescriptor{Name: "AMT", Type: 'N', Size: 8, Decimals: 2}
	got := parseField(fd, "  12.50")
	f, ok := got.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", got)
	}
	if f != 12.50 {
		t.Errorf("got %v, want 12.50", f)
	}
}

func TestParseField_NumericNoDecimals(t *testing.T) {
	fd := fieldDescriptor{Name: "CODE", Type: 'N', Size: 4, Decimals: 0}
	got := parseField(fd, "  42")
	f, ok := got.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", got)
	}
	if f != 42 {
		t.Errorf("got %v, want 42", f)
	}
}

func TestParseField_Date(t *testing.T) {
	fd := fieldDescriptor{Name: "DT", Type: 'D', Size: 8}
	got := parseField(fd, "20260101")
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T", got)
	}
	if s != "2026-01-01" {
		t.Errorf("got %q, want %q", s, "2026-01-01")
	}
}

func TestParseField_Logical(t *testing.T) {
	fd := fieldDescriptor{Name: "FLAG", Type: 'L', Size: 1}
	for _, raw := range []string{"T", "Y", "1", "t", "y"} {
		got := parseField(fd, raw)
		b, ok := got.(bool)
		if !ok || !b {
			t.Errorf("parseField(L, %q) = %v, want true", raw, got)
		}
	}
	for _, raw := range []string{"F", "N", "0", " ", ""} {
		got := parseField(fd, raw)
		b, ok := got.(bool)
		if !ok || b {
			t.Errorf("parseField(L, %q) = %v, want false", raw, got)
		}
	}
}
