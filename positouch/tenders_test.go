package positouch

import (
	"testing"
)

func TestReadTenders_Primary(t *testing.T) {
	dir := t.TempDir()

	fields := []fieldSpec{
		{name: "STORE", typ: 'C', size: 4},
		{name: "CODE", typ: 'N', size: 3, decimals: 0},
		{name: "NAME", typ: 'C', size: 11},
	}
	rows := [][]string{
		{"01  ", " 1 ", "Cash       "},
		{"01  ", " 2 ", "Credit Card"},
	}
	writeTempDBF(t, dir, "NAMEPAY.DBF", buildDBF(fields, rows))

	tenders, err := ReadTenders(dir)
	if err != nil {
		t.Fatalf("ReadTenders: %v", err)
	}
	if len(tenders) != 2 {
		t.Fatalf("got %d tenders, want 2", len(tenders))
	}
	if tenders[0].Code != 1 || tenders[0].Name != "Cash" {
		t.Errorf("tenders[0] = %+v, unexpected", tenders[0])
	}
	if tenders[0].Store != "01" {
		t.Errorf("tenders[0].Store = %q, want %q", tenders[0].Store, "01")
	}
}

func TestReadTenders_NamesFallback(t *testing.T) {
	dir := t.TempDir()

	// NAMES.DBF with CC and PY rows; only PY rows should be returned.
	fields := []fieldSpec{
		{name: "STORE", typ: 'C', size: 4},
		{name: "CODE", typ: 'C', size: 8},
		{name: "NAME", typ: 'C', size: 11},
	}
	rows := [][]string{
		{"01  ", "CC001   ", "Bar        "},
		{"01  ", "PY001   ", "Cash       "},
		{"01  ", "PY002   ", "Credit Card"},
	}
	writeTempDBF(t, dir, "NAMES.DBF", buildDBF(fields, rows))

	tenders, err := ReadTenders(dir)
	if err != nil {
		t.Fatalf("ReadTenders fallback: %v", err)
	}
	if len(tenders) != 2 {
		t.Fatalf("got %d tenders, want 2", len(tenders))
	}
	if tenders[0].Code != 1 || tenders[1].Code != 2 {
		t.Errorf("unexpected codes: %v, %v", tenders[0].Code, tenders[1].Code)
	}
}
