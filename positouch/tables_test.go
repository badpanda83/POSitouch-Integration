package positouch

import (
	"testing"
)

func TestReadTables_Primary(t *testing.T) {
	dir := t.TempDir()

	fields := []fieldSpec{
		{name: "STORE", typ: 'C', size: 4},
		{name: "TABLE", typ: 'N', size: 4, decimals: 0},
		{name: "COST_CENTR", typ: 'N', size: 3, decimals: 0},
	}
	rows := [][]string{
		{"01  ", "  10", "  1"},
		{"01  ", "  11", "  1"},
		{"01  ", "  10", "  1"}, // duplicate — should be deduplicated
	}
	writeTempDBF(t, dir, "CHKHDR.DBF", buildDBF(fields, rows))

	tables, err := ReadTables(dir)
	if err != nil {
		t.Fatalf("ReadTables: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("got %d tables, want 2 (deduplication)", len(tables))
	}
	if tables[0].Number != 10 {
		t.Errorf("tables[0].Number = %d, want 10", tables[0].Number)
	}
	if tables[0].Store != "01" {
		t.Errorf("tables[0].Store = %q, want %q", tables[0].Store, "01")
	}
}

func TestReadTables_Fallback(t *testing.T) {
	dir := t.TempDir()

	fields := []fieldSpec{
		{name: "STORE", typ: 'C', size: 4},
		{name: "TABLE_NO", typ: 'N', size: 4, decimals: 0},
		{name: "COSTCENTER", typ: 'N', size: 3, decimals: 0},
	}
	rows := [][]string{
		{"01  ", "   5", "  2"},
	}
	writeTempDBF(t, dir, "CHECK.DBF", buildDBF(fields, rows))

	tables, err := ReadTables(dir)
	if err != nil {
		t.Fatalf("ReadTables fallback: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}
	if tables[0].Number != 5 {
		t.Errorf("tables[0].Number = %d, want 5", tables[0].Number)
	}
	if tables[0].CostCenter != 2 {
		t.Errorf("tables[0].CostCenter = %d, want 2", tables[0].CostCenter)
	}
}
