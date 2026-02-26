package dbf

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

// buildDBF constructs a minimal valid dBASE III/IV byte stream for testing.
// fields is a list of (name, type, length, decimals) tuples.
// rows is a list of raw record byte slices (excluding the deletion flag byte).
func buildDBF(fields []fieldDescriptor, rows [][]byte) []byte {
	numFields := len(fields)
	headerSize := uint16(32 + numFields*32 + 1) // +1 for terminator byte

	// Calculate record size: 1 (deletion flag) + sum of field lengths.
	var recSize uint16 = 1
	for _, f := range fields {
		recSize += uint16(f.Length)
	}

	buf := &bytes.Buffer{}

	// --- 32-byte header ---
	hdr := make([]byte, 32)
	hdr[0] = 0x03                                           // version: dBASE III
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(rows))) // record count
	binary.LittleEndian.PutUint16(hdr[8:10], headerSize)
	binary.LittleEndian.PutUint16(hdr[10:12], recSize)
	buf.Write(hdr)

	// --- Field descriptors (32 bytes each) ---
	for _, f := range fields {
		fd := make([]byte, 32)
		copy(fd[0:], f.Name)
		fd[11] = f.Type
		fd[16] = f.Length
		fd[17] = f.Decimals
		buf.Write(fd)
	}
	// --- Header terminator ---
	buf.WriteByte(0x0D)

	// --- Records ---
	for _, row := range rows {
		buf.WriteByte(0x20) // deletion flag: space = active
		buf.Write(row)
	}

	return buf.Bytes()
}

// padRight pads s with spaces to length n.
func padRight(s string, n int) []byte {
	b := make([]byte, n)
	copy(b, s)
	for i := len(s); i < n; i++ {
		b[i] = ' '
	}
	return b
}

func TestReadCharacterField(t *testing.T) {
	fields := []fieldDescriptor{
		{Name: "NAME", Type: 'C', Length: 10},
	}
	row := padRight("Hello", 10)
	data := buildDBF(fields, [][]byte{row})

	records, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0]["NAME"] != "Hello" {
		t.Errorf("NAME = %q, want %q", records[0]["NAME"], "Hello")
	}
}

func TestReadNumericIntField(t *testing.T) {
	fields := []fieldDescriptor{
		{Name: "CODE", Type: 'N', Length: 5, Decimals: 0},
	}
	row := padRight("42", 5)
	data := buildDBF(fields, [][]byte{row})

	records, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if records[0]["CODE"] != 42 {
		t.Errorf("CODE = %v (%T), want 42 (int)", records[0]["CODE"], records[0]["CODE"])
	}
}

func TestReadNumericFloatField(t *testing.T) {
	fields := []fieldDescriptor{
		{Name: "PRICE", Type: 'N', Length: 10, Decimals: 2},
	}
	row := padRight("9.99", 10)
	data := buildDBF(fields, [][]byte{row})

	records, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	got, ok := records[0]["PRICE"].(float64)
	if !ok {
		t.Fatalf("PRICE type = %T, want float64", records[0]["PRICE"])
	}
	if got != 9.99 {
		t.Errorf("PRICE = %v, want 9.99", got)
	}
}

func TestReadDateField(t *testing.T) {
	fields := []fieldDescriptor{
		{Name: "HIRED", Type: 'D', Length: 8},
	}
	row := []byte("20240115")
	data := buildDBF(fields, [][]byte{row})

	records, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	got, ok := records[0]["HIRED"].(time.Time)
	if !ok {
		t.Fatalf("HIRED type = %T, want time.Time", records[0]["HIRED"])
	}
	want := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("HIRED = %v, want %v", got, want)
	}
}

func TestReadLogicalField(t *testing.T) {
	fields := []fieldDescriptor{
		{Name: "ACTIVE", Type: 'L', Length: 1},
	}
	for _, tt := range []struct {
		raw  string
		want interface{}
	}{
		{"T", true},
		{"Y", true},
		{"F", false},
		{"N", false},
		{" ", nil},
	} {
		row := []byte(tt.raw)
		data := buildDBF(fields, [][]byte{row})
		records, err := Read(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Read() error: %v", err)
		}
		if records[0]["ACTIVE"] != tt.want {
			t.Errorf("raw=%q: ACTIVE = %v, want %v", tt.raw, records[0]["ACTIVE"], tt.want)
		}
	}
}

func TestSkipsDeletedRecords(t *testing.T) {
	// Two records: first deleted, second active.
	var data []byte
	hdrSize := uint16(32 + 1*32 + 1)
	recSize := uint16(1 + 2)

	hdr := make([]byte, 32)
	hdr[0] = 0x03
	binary.LittleEndian.PutUint32(hdr[4:8], 2)
	binary.LittleEndian.PutUint16(hdr[8:10], hdrSize)
	binary.LittleEndian.PutUint16(hdr[10:12], recSize)
	data = append(data, hdr...)

	fd := make([]byte, 32)
	copy(fd[0:], "ID")
	fd[11] = 'N'
	fd[16] = 2
	fd[17] = 0
	data = append(data, fd...)
	data = append(data, 0x0D)

	// Deleted record (flag = '*')
	data = append(data, '*')
	data = append(data, []byte(" 1")...)
	// Active record
	data = append(data, 0x20)
	data = append(data, []byte(" 2")...)

	records, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 non-deleted record, got %d", len(records))
	}
	if records[0]["ID"] != 2 {
		t.Errorf("ID = %v, want 2", records[0]["ID"])
	}
}
