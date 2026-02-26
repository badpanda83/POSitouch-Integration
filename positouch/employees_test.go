package positouch

import (
	"testing"
)

func TestReadEmployees_UsersOnly(t *testing.T) {
	dir := t.TempDir()

	fields := []fieldSpec{
		{name: "STORE", typ: 'C', size: 4},
		{name: "USER_NBR", typ: 'N', size: 4, decimals: 0},
		{name: "NAME_LAST", typ: 'C', size: 20},
		{name: "NAME_FIRST", typ: 'C', size: 20},
		{name: "TYPE", typ: 'N', size: 2, decimals: 0},
		{name: "MAGCARD_ID", typ: 'N', size: 8, decimals: 0},
	}
	rows := [][]string{
		{"01  ", "  42", "Smith               ", "Jane                ", " 1", "       0"},
	}
	writeTempDBF(t, dir, "USERS.DBF", buildDBF(fields, rows))

	emps, err := ReadEmployees(dir, t.TempDir())
	if err != nil {
		t.Fatalf("ReadEmployees: %v", err)
	}
	if len(emps) != 1 {
		t.Fatalf("got %d employees, want 1", len(emps))
	}
	e := emps[0]
	if e.Number != 42 {
		t.Errorf("Number = %d, want 42", e.Number)
	}
	if e.LastName != "Smith" {
		t.Errorf("LastName = %q, want %q", e.LastName, "Smith")
	}
	if e.FirstName != "Jane" {
		t.Errorf("FirstName = %q, want %q", e.FirstName, "Jane")
	}
	if e.Store != "01" {
		t.Errorf("Store = %q, want %q", e.Store, "01")
	}
}

func TestReadEmployees_WithEmpFile(t *testing.T) {
	dbfDir := t.TempDir()
	scDir := t.TempDir()

	userFields := []fieldSpec{
		{name: "STORE", typ: 'C', size: 4},
		{name: "USER_NBR", typ: 'N', size: 4, decimals: 0},
		{name: "NAME_LAST", typ: 'C', size: 20},
		{name: "NAME_FIRST", typ: 'C', size: 20},
		{name: "TYPE", typ: 'N', size: 2, decimals: 0},
		{name: "MAGCARD_ID", typ: 'N', size: 8, decimals: 0},
	}
	userRows := [][]string{
		{"01  ", "  10", "Doe                 ", "John                ", " 2", "       0"},
	}
	writeTempDBF(t, dbfDir, "USERS.DBF", buildDBF(userFields, userRows))

	empFields := []fieldSpec{
		{name: "EMP_NUMBER", typ: 'N', size: 4, decimals: 0},
		{name: "EMP_STATUS", typ: 'C', size: 2},
	}
	empRows := [][]string{
		{"  10", "F "},
	}
	writeTempDBF(t, scDir, "EMPFILE.DBF", buildDBF(empFields, empRows))

	emps, err := ReadEmployees(dbfDir, scDir)
	if err != nil {
		t.Fatalf("ReadEmployees with EMPFILE: %v", err)
	}
	if len(emps) != 1 {
		t.Fatalf("got %d employees, want 1", len(emps))
	}
	if emps[0].Status != "F" {
		t.Errorf("Status = %q, want %q", emps[0].Status, "F")
	}
}
