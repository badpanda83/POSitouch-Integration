package dbf_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// buildDBF creates a minimal valid dBASE III DBF byte stream in memory for testing.
func buildDBF(fields []struct {
	name     string
	typ      byte
	length   uint8
	decimals uint8
}, rows [][]string) []byte {
	// Compute header size: 32 (file header) + 32*len(fields) + 1 (terminator).
	numFields := len(fields)
	headerSize := uint16(32 + numFields*32 + 1)

	// Record size: 1 (deletion flag) + sum of field lengths.
	var recSize uint16 = 1
	for _, f := range fields {
		recSize += uint16(f.length)
	}

	buf := new(bytes.Buffer)

	// ── File header (32 bytes) ────────────────────────────────────────────
	fileHeader := make([]byte, 32)
	fileHeader[0] = 0x03 // version: dBASE III
	binary.LittleEndian.PutUint32(fileHeader[4:8], uint32(len(rows)))
	binary.LittleEndian.PutUint16(fileHeader[8:10], headerSize)
	binary.LittleEndian.PutUint16(fileHeader[10:12], recSize)
	buf.Write(fileHeader)

	// ── Field descriptors (32 bytes each) ────────────────────────────────
	for _, f := range fields {
		desc := make([]byte, 32)
		copy(desc[0:11], []byte(f.name))
		desc[11] = f.typ
		desc[16] = f.length
		desc[17] = f.decimals
		buf.Write(desc)
	}
	// Header terminator.
	buf.WriteByte(0x0D)

	// ── Data records ─────────────────────────────────────────────────────
	for _, row := range rows {
		buf.WriteByte(' ') // active record
		for i, f := range fields {
			cell := make([]byte, f.length)
			for j := range cell {
				cell[j] = ' '
			}
			if i < len(row) {
				src := []byte(row[i])
				if len(src) > int(f.length) {
					src = src[:f.length]
				}
				copy(cell, src)
			}
			buf.Write(cell)
		}
	}

	return buf.Bytes()
}

func TestRead_BasicFields(t *testing.T) {
	fields := []struct {
		name     string
		typ      byte
		length   uint8
		decimals uint8
	}{
		{"STORE", 'C', 4, 0},
		{"CODE", 'N', 2, 0},
		{"NAME", 'C', 20, 0},
	}
	rows := [][]string{
		{"0001", "1 ", "Dining Room         "},
		{"0001", "2 ", "Bar                 "},
	}

	data := buildDBF(fields, rows)
	table, err := dbf.Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	if len(table.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(table.Records))
	}

	// First record.
	rec := table.Records[0]
	if got := dbf.FieldString(rec, "STORE"); got != "0001" {
		t.Errorf("STORE: want %q, got %q", "0001", got)
	}
	code, ok := dbf.FieldInt(rec, "CODE")
	if !ok || code != 1 {
		t.Errorf("CODE: want 1/true, got %d/%v", code, ok)
	}
	if got := dbf.FieldString(rec, "NAME"); got != "Dining Room" {
		t.Errorf("NAME: want %q, got %q", "Dining Room", got)
	}

	// Second record.
	rec2 := table.Records[1]
	if got := dbf.FieldString(rec2, "NAME"); got != "Bar" {
		t.Errorf("NAME: want %q, got %q", "Bar", got)
	}
}

func TestRead_DeletedRecord(t *testing.T) {
	// Build raw bytes manually so we can mark one record as deleted.
	headerSize := uint16(32 + 1*32 + 1)
	recSize := uint16(1 + 10)
	fileHeader := make([]byte, 32)
	fileHeader[0] = 0x03
	binary.LittleEndian.PutUint32(fileHeader[4:8], 2)
	binary.LittleEndian.PutUint16(fileHeader[8:10], headerSize)
	binary.LittleEndian.PutUint16(fileHeader[10:12], recSize)

	buf := new(bytes.Buffer)
	buf.Write(fileHeader)
	// Field descriptor.
	desc := make([]byte, 32)
	copy(desc[0:11], []byte("NAME"))
	desc[11] = 'C'
	desc[16] = 10
	buf.Write(desc)
	buf.WriteByte(0x0D)
	// Active record.
	buf.WriteByte(' ')
	buf.Write([]byte("Active    "))
	// Deleted record.
	buf.WriteByte('*')
	buf.Write([]byte("Deleted   "))

	table, err := dbf.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(table.Records) != 1 {
		t.Errorf("expected 1 active record, got %d", len(table.Records))
	}
	if got := dbf.FieldString(table.Records[0], "NAME"); got != "Active" {
		t.Errorf("expected %q, got %q", "Active", got)
	}
}

func TestRead_LogicalField(t *testing.T) {
	fields := []struct {
		name     string
		typ      byte
		length   uint8
		decimals uint8
	}{
		{"FLAG", 'L', 1, 0},
	}
	rows := [][]string{{"T"}, {"F"}, {"Y"}, {"N"}}

	data := buildDBF(fields, rows)
	table, err := dbf.Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(table.Records) != 4 {
		t.Fatalf("expected 4 records, got %d", len(table.Records))
	}
	want := []string{"true", "false", "true", "false"}
	for i, rec := range table.Records {
		if got := rec["FLAG"]; got != want[i] {
			t.Errorf("record %d FLAG: want %q, got %q", i, want[i], got)
		}
	}
}

func TestFieldInt_MissingField(t *testing.T) {
	rec := dbf.Record{"NAME": "test"}
	n, ok := dbf.FieldInt(rec, "MISSING")
	if ok || n != 0 {
		t.Errorf("expected (0, false), got (%d, %v)", n, ok)
	}
}
