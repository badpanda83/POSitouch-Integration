package positouch

import (
	"testing"
)

func TestReadOrderTypes(t *testing.T) {
	dir := t.TempDir()

	fields := []fieldSpec{
		{name: "STORE", typ: 'C', size: 4},
		{name: "MENU_NUM", typ: 'N', size: 3, decimals: 0},
		{name: "MENU_TITLE", typ: 'C', size: 20},
		{name: "FF_ORD_T", typ: 'N', size: 2, decimals: 0},
	}
	rows := [][]string{
		{"01  ", "  1", "Dine In             ", " 0"},
		{"01  ", "  2", "Take Out            ", " 1"},
	}
	writeTempDBF(t, dir, "MENUS.DBF", buildDBF(fields, rows))

	orderTypes, err := ReadOrderTypes(dir)
	if err != nil {
		t.Fatalf("ReadOrderTypes: %v", err)
	}
	if len(orderTypes) != 2 {
		t.Fatalf("got %d order types, want 2", len(orderTypes))
	}
	ot := orderTypes[0]
	if ot.MenuNumber != 1 {
		t.Errorf("MenuNumber = %d, want 1", ot.MenuNumber)
	}
	if ot.Title != "Dine In" {
		t.Errorf("Title = %q, want %q", ot.Title, "Dine In")
	}
	if ot.FFOrderType != 0 {
		t.Errorf("FFOrderType = %d, want 0", ot.FFOrderType)
	}
	if ot.Store != "01" {
		t.Errorf("Store = %q, want %q", ot.Store, "01")
	}
}

func TestReadOrderTypes_NotFound(t *testing.T) {
	_, err := ReadOrderTypes(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing MENUS.DBF, got nil")
	}
}
