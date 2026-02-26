package dbf

import (
"bytes"
"encoding/binary"
"testing"
)

// buildDBF constructs a minimal valid DBF byte stream in memory.
func buildDBF(fields []fieldDescriptor, rows [][]string) []byte {
headerSize := uint16(32 + len(fields)*32 + 1) // +1 for terminator
recordSize := uint16(1)                         // deletion flag
for _, f := range fields {
recordSize += uint16(f.Length)
}

buf := &bytes.Buffer{}

// Header (32 bytes).
header := make([]byte, 32)
header[0] = 0x03 // version
binary.LittleEndian.PutUint32(header[4:], uint32(len(rows)))
binary.LittleEndian.PutUint16(header[8:], headerSize)
binary.LittleEndian.PutUint16(header[10:], recordSize)
buf.Write(header)

// Field descriptors (32 bytes each).
for _, f := range fields {
fd := make([]byte, 32)
copy(fd[0:11], []byte(f.Name))
fd[11] = f.Type
fd[16] = f.Length
fd[17] = f.DecimalCount
buf.Write(fd)
}
buf.WriteByte(0x0D) // terminator

// Records.
for _, row := range rows {
buf.WriteByte(0x20) // not deleted
for i, f := range fields {
val := ""
if i < len(row) {
val = row[i]
}
padded := make([]byte, f.Length)
for j := range padded {
padded[j] = ' '
}
copy(padded, []byte(val))
buf.Write(padded)
}
}
buf.WriteByte(0x1A) // EOF marker

return buf.Bytes()
}

func TestReadRecords_Basic(t *testing.T) {
fields := []fieldDescriptor{
{Name: "CODE", Type: 'N', Length: 2},
{Name: "NAME", Type: 'C', Length: 10},
}
rows := [][]string{
{"1 ", "Alpha     "},
{"2 ", "Beta      "},
}

data := buildDBF(fields, rows)
records, err := readRecords(bytes.NewReader(data))
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(records) != 2 {
t.Fatalf("expected 2 records, got %d", len(records))
}
if records[0]["CODE"] != "1" {
t.Errorf("CODE: expected '1', got %q", records[0]["CODE"])
}
if records[0]["NAME"] != "Alpha" {
t.Errorf("NAME: expected 'Alpha', got %q", records[0]["NAME"])
}
}

func TestReadRecords_DeletedSkipped(t *testing.T) {
// Build manually with one deleted record.
headerSize := uint16(32 + 1*32 + 1)
recordSize := uint16(1 + 2)
buf := &bytes.Buffer{}
header := make([]byte, 32)
header[0] = 0x03
binary.LittleEndian.PutUint32(header[4:], 2)
binary.LittleEndian.PutUint16(header[8:], headerSize)
binary.LittleEndian.PutUint16(header[10:], recordSize)
buf.Write(header)
fd := make([]byte, 32)
copy(fd[0:11], "CODE")
fd[11] = 'N'
fd[16] = 2
buf.Write(fd)
buf.WriteByte(0x0D)
// record 1: deleted
buf.WriteByte(0x2A)
buf.WriteString("1 ")
// record 2: active
buf.WriteByte(0x20)
buf.WriteString("2 ")
buf.WriteByte(0x1A)

records, err := readRecords(bytes.NewReader(buf.Bytes()))
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(records) != 1 {
t.Fatalf("expected 1 record after skipping deleted, got %d", len(records))
}
if records[0]["CODE"] != "2" {
t.Errorf("CODE: expected '2', got %q", records[0]["CODE"])
}
}

func TestReadRecords_LogicalField(t *testing.T) {
fields := []fieldDescriptor{
{Name: "FLAG", Type: 'L', Length: 1},
}
rows := [][]string{{"T"}, {"F"}, {"Y"}, {"N"}}
data := buildDBF(fields, rows)
records, err := readRecords(bytes.NewReader(data))
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
expected := []string{"true", "false", "true", "false"}
for i, rec := range records {
if rec["FLAG"] != expected[i] {
t.Errorf("record %d FLAG: expected %q, got %q", i, expected[i], rec["FLAG"])
}
}
}

func TestReadFile_NotFound(t *testing.T) {
_, err := ReadFile("/nonexistent/path/file.dbf")
if err == nil {
t.Fatal("expected error for missing file, got nil")
}
}
