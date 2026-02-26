package positouch

import (
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Employee represents a POSitouch user/employee record.
type Employee struct {
	UserNumber  int64  `json:"user_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Type        int64  `json:"type"`
	MagCardID   int64  `json:"mag_card_id,omitempty"`
	TblAssign   int64  `json:"tbl_assign,omitempty"`
	CCTable1    int64  `json:"cc_table_1,omitempty"`
	CCTable2    int64  `json:"cc_table_2,omitempty"`
	CCTable3    int64  `json:"cc_table_3,omitempty"`
	CCTable4    int64  `json:"cc_table_4,omitempty"`
	Store       string `json:"store"`
}

// LoadEmployees reads employee records from USERS.DBF (produced by POSIDBFW).
// If USERS.DBF is not found it tries EMPFILE.DBF (produced by TAW EXPORT).
func LoadEmployees(dbfDir string) ([]Employee, error) {
	primary := filepath.Join(dbfDir, "USERS.DBF")
	r, err := dbf.Open(primary)
	if err == nil {
		return parseUsers(r.Records()), nil
	}

	// Fallback: EMPFILE.DBF in the same directory or SC directory.
	fallback := filepath.Join(dbfDir, "EMPFILE.DBF")
	r, err = dbf.Open(fallback)
	if err != nil {
		return nil, err
	}
	return parseEmpFile(r.Records()), nil
}

func parseUsers(records []dbf.Record) []Employee {
	out := make([]Employee, 0, len(records))
	for _, rec := range records {
		out = append(out, Employee{
			UserNumber: rec.GetInt("USER_NBR"),
			FirstName:  rec.GetString("NAME_FIRST"),
			LastName:   rec.GetString("NAME_LAST"),
			Type:       rec.GetInt("TYPE"),
			MagCardID:  rec.GetInt("MAGCARD_ID"),
			TblAssign:  rec.GetInt("TBL_ASSGN"),
			CCTable1:   rec.GetInt("CC_TABLE_1"),
			CCTable2:   rec.GetInt("CC_TABLE_2"),
			CCTable3:   rec.GetInt("CC_TABLE_3"),
			CCTable4:   rec.GetInt("CC_TABLE_4"),
			Store:      rec.GetString("STORE"),
		})
	}
	return out
}

func parseEmpFile(records []dbf.Record) []Employee {
	out := make([]Employee, 0, len(records))
	for _, rec := range records {
		out = append(out, Employee{
			UserNumber: rec.GetInt("EMP_NUMBER"),
			FirstName:  rec.GetString("FIRST_NAME"),
			LastName:   rec.GetString("LAST_NAME"),
			Store:      rec.GetString("STORE"),
		})
	}
	return out
}
