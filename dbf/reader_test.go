package dbf_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// buildDBF constructs a minimal valid dBASE III DBF byte slice with the
// supplied field descriptors and row data.
//
// fields is a slice of [name, type, length, decimals] tuples encoded as
// structs for clarity.
func buildTestDBF(fields []testField, rows [][]string) []byte {
	// header = 32 bytes
	// field descriptors = 32 bytes each + 1 terminator byte
	// records = (1 deletion flag + sum(field lengths)) * numRows

	recordLen := 1 // deletion flag
	for _, f := range fields {
		recordLen += f.Length
	}
	headerBytes := 32 + len(fields)*32 + 1 // +1 for 0x0D terminator

	var buf bytes.Buffer

	// --- DBF header ---
	hdr := make([]byte, 32)
	hdr[0] = 0x03 // dBASE III
	hdr[1] = 26   // year
	hdr[2] = 2    // month
	hdr[3] = 26   // day
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(rows)))
	binary.LittleEndian.PutUint16(hdr[8:10], uint16(headerBytes))
	binary.LittleEndian.PutUint16(hdr[10:12], uint16(recordLen))
	buf.Write(hdr)

	// --- field descriptors ---
	for _, f := range fields {
		fd := make([]byte, 32)
		copy(fd[0:11], []byte(f.Name))
		fd[11] = f.Type
		fd[16] = byte(f.Length)
		fd[17] = byte(f.Decimals)
		buf.Write(fd)
	}
	buf.WriteByte(0x0D) // terminator

	// --- records ---
	for _, row := range rows {
		buf.WriteByte(0x20) // active record
		for i, f := range fields {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			// Pad or truncate to field length.
			padded := make([]byte, f.Length)
			for j := range padded {
				padded[j] = ' '
			}
			copy(padded, []byte(val))
			buf.Write(padded)
		}
	}

	return buf.Bytes()
}

type testField struct {
	Name     string
	Type     byte
	Length   int
	Decimals int
}

func TestRecordParsing(t *testing.T) {
	fields := []testField{
		{"STORE", 'C', 4, 0},
		{"CODE", 'N', 2, 0},
		{"NAME", 'C', 20, 0},
	}
	rows := [][]string{
		{"0001", "1", "Dining Room         "},
		{"0001", "2", "Bar                 "},
	}

	data := buildTestDBF(fields, rows)

	// Write to a temp file via a bytes.Reader so we can call the open path.
	// Since we can't mock Open directly, use the exported read-like path via
	// writing to a temp file.
	importPath := t.TempDir() + "/TEST.DBF"
	if err := writeFile(importPath, data); err != nil {
		t.Fatalf("failed to write temp DBF: %v", err)
	}

	r, err := dbf.Open(importPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	records := r.Records()
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	if got := records[0].GetString("STORE"); got != "0001" {
		t.Errorf("STORE = %q, want %q", got, "0001")
	}
	if got := records[0].GetInt("CODE"); got != 1 {
		t.Errorf("CODE = %d, want 1", got)
	}
	if got := records[0].GetString("NAME"); got != "Dining Room" {
		t.Errorf("NAME = %q, want %q", got, "Dining Room")
	}
	if got := records[1].GetInt("CODE"); got != 2 {
		t.Errorf("second CODE = %d, want 2", got)
	}
}

func TestDeletedRecords(t *testing.T) {
	recordLen := 1 + 2
	headerBytes := 32 + 1*32 + 1

	var buf bytes.Buffer
	hdr := make([]byte, 32)
	binary.LittleEndian.PutUint32(hdr[4:8], 2)
	binary.LittleEndian.PutUint16(hdr[8:10], uint16(headerBytes))
	binary.LittleEndian.PutUint16(hdr[10:12], uint16(recordLen))
	buf.Write(hdr)

	fd := make([]byte, 32)
	copy(fd[0:11], "CODE")
	fd[11] = 'N'
	fd[16] = 2
	buf.Write(fd)
	buf.WriteByte(0x0D)

	// first record: deleted
	buf.WriteByte(0x2A)
	buf.WriteString(" 1")
	// second record: active
	buf.WriteByte(0x20)
	buf.WriteString(" 2")

	path := t.TempDir() + "/DEL.DBF"
	if err := writeFile(path, buf.Bytes()); err != nil {
		t.Fatalf("write temp: %v", err)
	}

	r, err := dbf.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	recs := r.Records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 active record, got %d", len(recs))
	}
	if got := recs[0].GetInt("CODE"); got != 2 {
		t.Errorf("CODE = %d, want 2", got)
	}
}

func TestLogicalField(t *testing.T) {
	fields := []testField{
		{"FLAG", 'L', 1, 0},
	}
	rows := [][]string{
		{"T"},
		{"F"},
		{"Y"},
		{"N"},
	}
	data := buildTestDBF(fields, rows)
	path := t.TempDir() + "/LOGICAL.DBF"
	if err := writeFile(path, data); err != nil {
		t.Fatal(err)
	}
	r, err := dbf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recs := r.Records()
	if len(recs) != 4 {
		t.Fatalf("expected 4 records, got %d", len(recs))
	}
	want := []bool{true, false, true, false}
	for i, w := range want {
		if got := recs[i].GetBool("FLAG"); got != w {
			t.Errorf("record %d FLAG = %v, want %v", i, got, w)
		}
	}
}

func TestNumericDecimalField(t *testing.T) {
	fields := []testField{
		{"PRICE", 'N', 8, 2},
	}
	rows := [][]string{
		{"   12.50"},
	}
	data := buildTestDBF(fields, rows)
	path := t.TempDir() + "/NUMERIC.DBF"
	if err := writeFile(path, data); err != nil {
		t.Fatal(err)
	}
	r, err := dbf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recs := r.Records()
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if got := recs[0].GetFloat("PRICE"); got != 12.50 {
		t.Errorf("PRICE = %v, want 12.50", got)
	}
}

// writeFile writes data to path, creating it.
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}
