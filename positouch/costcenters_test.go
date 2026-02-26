package positouch

import (
	"testing"
)

func TestReadCostCenters_Primary(t *testing.T) {
	dir := t.TempDir()

	fields := []fieldSpec{
		{name: "STORE", typ: 'C', size: 4},
		{name: "CODE", typ: 'N', size: 3, decimals: 0},
		{name: "NAME", typ: 'C', size: 10},
	}
	rows := [][]string{
		{"01  ", " 1 ", "Bar       "},
		{"01  ", " 2 ", "Kitchen   "},
	}
	writeTempDBF(t, dir, "NAMECC.DBF", buildDBF(fields, rows))

	centers, err := ReadCostCenters(dir)
	if err != nil {
		t.Fatalf("ReadCostCenters: %v", err)
	}
	if len(centers) != 2 {
		t.Fatalf("got %d centers, want 2", len(centers))
	}
	if centers[0].Code != 1 || centers[0].Name != "Bar" {
		t.Errorf("centers[0] = %+v, unexpected", centers[0])
	}
	if centers[0].Store != "01" {
		t.Errorf("centers[0].Store = %q, want %q", centers[0].Store, "01")
	}
}

func TestReadCostCenters_NamesFallback(t *testing.T) {
	dir := t.TempDir()

	// NAMES.DBF with CC and PY rows; only CC rows should be returned.
	fields := []fieldSpec{
		{name: "STORE", typ: 'C', size: 4},
		{name: "CODE", typ: 'C', size: 8},
		{name: "NAME", typ: 'C', size: 10},
	}
	rows := [][]string{
		{"01  ", "CC001   ", "Bar       "},
		{"01  ", "PY001   ", "Cash      "},
		{"01  ", "CC002   ", "Kitchen   "},
	}
	writeTempDBF(t, dir, "NAMES.DBF", buildDBF(fields, rows))

	centers, err := ReadCostCenters(dir)
	if err != nil {
		t.Fatalf("ReadCostCenters fallback: %v", err)
	}
	if len(centers) != 2 {
		t.Fatalf("got %d centers, want 2", len(centers))
	}
	if centers[0].Code != 1 || centers[1].Code != 2 {
		t.Errorf("unexpected codes: %v, %v", centers[0].Code, centers[1].Code)
	}
}
