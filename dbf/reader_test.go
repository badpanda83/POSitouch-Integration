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
