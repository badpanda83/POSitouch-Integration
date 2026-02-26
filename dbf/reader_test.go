package dbf_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// buildTestDBF creates a minimal valid dBASE III byte slice with two fields
// (NAME C(10) and CODE N(3)) and two records — one active, one deleted.
func buildTestDBF() []byte {
	const (
		numFields  = 2
		headerSize = 32 + numFields*32 + 1 // 97
		recordSize = 1 + 10 + 3            // deletion flag + NAME + CODE
		numRecords = 2
	)

	buf := new(bytes.Buffer)

	// File header (32 bytes)
	buf.WriteByte(0x03) // dBASE III
	buf.Write([]byte{26, 2, 26})
	rc := make([]byte, 4)
	binary.LittleEndian.PutUint32(rc, numRecords)
	buf.Write(rc)
	hs := make([]byte, 2)
	binary.LittleEndian.PutUint16(hs, headerSize)
	buf.Write(hs)
	rs := make([]byte, 2)
	binary.LittleEndian.PutUint16(rs, recordSize)
	buf.Write(rs)
	buf.Write(make([]byte, 20)) // reserved

	// Field 1: NAME C(10)
	f1 := make([]byte, 11)
	copy(f1, "NAME")
	buf.Write(f1)
	buf.WriteByte('C')
	buf.Write(make([]byte, 4))
	buf.WriteByte(10)
	buf.WriteByte(0)
	buf.Write(make([]byte, 14))

	// Field 2: CODE N(3)
	f2 := make([]byte, 11)
	copy(f2, "CODE")
	buf.Write(f2)
	buf.WriteByte('N')
	buf.Write(make([]byte, 4))
	buf.WriteByte(3)
	buf.WriteByte(0)
	buf.Write(make([]byte, 14))

	// Header terminator
	buf.WriteByte(0x0D)

	// Pad header to exactly headerSize bytes
	for buf.Len() < headerSize {
		buf.WriteByte(0x00)
	}

	// Record 1 (active): NAME="Bar&Grill ", CODE=" 42"
	buf.WriteByte(0x20)
	nameVal := make([]byte, 10)
	copy(nameVal, "Bar&Grill ")
	buf.Write(nameVal)
	buf.WriteString(" 42")

	// Record 2 (deleted)
	buf.WriteByte(0x2A)
	buf.Write(make([]byte, 10))
	buf.WriteString("  1")

	return buf.Bytes()
}

func TestReadFile_ActiveRecordsOnly(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.dbf")
	if err := os.WriteFile(tmp, buildTestDBF(), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	records, err := dbf.ReadFile(tmp)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	rec := records[0]

	nameVal, ok := rec["NAME"].(string)
	if !ok {
		t.Fatalf("NAME field is not a string: %T", rec["NAME"])
	}
	if nameVal != "Bar&Grill" {
		t.Errorf("NAME: got %q, want %q", nameVal, "Bar&Grill")
	}

	codeVal, ok := rec["CODE"].(float64)
	if !ok {
		t.Fatalf("CODE field is not float64: %T", rec["CODE"])
	}
	if codeVal != 42 {
		t.Errorf("CODE: got %v, want 42", codeVal)
	}
}

func TestReadFile_NotFound(t *testing.T) {
	_, err := dbf.ReadFile(filepath.Join(t.TempDir(), "nonexistent.dbf"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// buildDBFWithDate creates a DBF with a DATE field D(8) for testing date parsing.
func buildDBFWithDate(dateVal string) []byte {
	const (
		headerSize = 32 + 32 + 1 // 65
		recordSize = 1 + 8
	)
	buf := new(bytes.Buffer)
	header := make([]byte, 32)
	header[0] = 0x03
	binary.LittleEndian.PutUint32(header[4:8], 1)
	binary.LittleEndian.PutUint16(header[8:10], headerSize)
	binary.LittleEndian.PutUint16(header[10:12], recordSize)
	buf.Write(header)

	fd := make([]byte, 32)
	copy(fd[0:11], "DATE")
	fd[11] = 'D'
	fd[16] = 8
	buf.Write(fd)
	buf.WriteByte(0x0D)

	buf.WriteByte(0x20)
	val := make([]byte, 8)
	copy(val, dateVal)
	buf.Write(val)
	return buf.Bytes()
}

func TestParseField_DateFormatted(t *testing.T) {
	data := buildDBFWithDate("20260101")
	tmp := filepath.Join(t.TempDir(), "date.dbf")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	records, err := dbf.ReadFile(tmp)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	got, ok := records[0]["DATE"].(string)
	if !ok {
		t.Fatalf("DATE is not string: %T", records[0]["DATE"])
	}
	if got != "2026-01-01" {
		t.Errorf("DATE: got %q, want %q", got, "2026-01-01")
	}
}

// buildDBFWithLogical creates a DBF with a LOGICAL field L(1).
func buildDBFWithLogical(val string) []byte {
	const (
		headerSize = 32 + 32 + 1
		recordSize = 1 + 1
	)
	buf := new(bytes.Buffer)
	header := make([]byte, 32)
	header[0] = 0x03
	binary.LittleEndian.PutUint32(header[4:8], 1)
	binary.LittleEndian.PutUint16(header[8:10], headerSize)
	binary.LittleEndian.PutUint16(header[10:12], recordSize)
	buf.Write(header)

	fd := make([]byte, 32)
	copy(fd[0:11], "FLAG")
	fd[11] = 'L'
	fd[16] = 1
	buf.Write(fd)
	buf.WriteByte(0x0D)

	buf.WriteByte(0x20)
	buf.WriteByte(val[0])
	return buf.Bytes()
}

func TestParseField_LogicalTrue(t *testing.T) {
	for _, ch := range []string{"T", "Y", "1"} {
		data := buildDBFWithLogical(ch)
		tmp := filepath.Join(t.TempDir(), "log.dbf")
		if err := os.WriteFile(tmp, data, 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		records, err := dbf.ReadFile(tmp)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		got, ok := records[0]["FLAG"].(bool)
		if !ok || !got {
			t.Errorf("FLAG=%q: got %v, want true", ch, records[0]["FLAG"])
		}
	}
}

func TestParseField_LogicalFalse(t *testing.T) {
	for _, ch := range []string{"F", "N"} {
		data := buildDBFWithLogical(ch)
		tmp := filepath.Join(t.TempDir(), "log.dbf")
		if err := os.WriteFile(tmp, data, 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		records, err := dbf.ReadFile(tmp)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		got, ok := records[0]["FLAG"].(bool)
		if !ok || got {
			t.Errorf("FLAG=%q: got %v, want false", ch, records[0]["FLAG"])
		}
	}
}
