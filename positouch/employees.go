package positouch

import (
	"fmt"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Employee represents one row from USERS.DBF.
type Employee struct {
	Store      string `json:"store"`
	UserNumber int    `json:"user_number"`
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	Type       int    `json:"type"`
	MagCardID  int    `json:"mag_card_id,omitempty"`
	TblAssign  int    `json:"tbl_assign,omitempty"`
	CCTable1   int    `json:"cc_table_1,omitempty"`
	CCTable2   int    `json:"cc_table_2,omitempty"`
	CCTable3   int    `json:"cc_table_3,omitempty"`
	CCTable4   int    `json:"cc_table_4,omitempty"`
}

// ReadEmployees opens USERS.DBF from dbfDir and returns all employees.
func ReadEmployees(dbfDir string) ([]Employee, error) {
	path := filepath.Join(dbfDir, "USERS.DBF")
	table, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("employees: %w", err)
	}

	result := make([]Employee, 0, len(table.Records))
	for _, rec := range table.Records {
		userNbr, _ := dbf.FieldInt(rec, "USER_NBR")
		empType, _ := dbf.FieldInt(rec, "TYPE")
		magCard, _ := dbf.FieldInt(rec, "MAGCARD_ID")
		tblAssign, _ := dbf.FieldInt(rec, "TBL_ASSGN")
		cc1, _ := dbf.FieldInt(rec, "CC_TABLE_1")
		cc2, _ := dbf.FieldInt(rec, "CC_TABLE_2")
		cc3, _ := dbf.FieldInt(rec, "CC_TABLE_3")
		cc4, _ := dbf.FieldInt(rec, "CC_TABLE_4")

		result = append(result, Employee{
			Store:      dbf.FieldString(rec, "STORE"),
			UserNumber: userNbr,
			LastName:   dbf.FieldString(rec, "NAME_LAST"),
			FirstName:  dbf.FieldString(rec, "NAME_FIRST"),
			Type:       empType,
			MagCardID:  magCard,
			TblAssign:  tblAssign,
			CCTable1:   cc1,
			CCTable2:   cc2,
			CCTable3:   cc3,
			CCTable4:   cc4,
		})
	}
	return result, nil
}
