package positouch

import (
	"fmt"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Table represents one unique table entry derived from CHKHDR.DBF / CHECK.DBF.
type Table struct {
	Store      string `json:"store"`
	TableNumber int   `json:"table_number"`
	CostCenter  int   `json:"cost_center"`
}

// ReadTables opens CHKHDR.DBF (falling back to CHECK.DBF) from dbfDir and
// returns unique table definitions.
func ReadTables(dbfDir string) ([]Table, error) {
	path := filepath.Join(dbfDir, "CHKHDR.DBF")
	table, err := dbf.Open(path)
	if err != nil {
		// Try alternate filename.
		alt := filepath.Join(dbfDir, "CHECK.DBF")
		table, err = dbf.Open(alt)
		if err != nil {
			return nil, fmt.Errorf("tables: %w", err)
		}
	}

	seen := make(map[int]bool)
	result := make([]Table, 0)
	for _, rec := range table.Records {
		tblNum, ok := dbf.FieldInt(rec, "TABLE")
		if !ok || tblNum == 0 {
			continue
		}
		if seen[tblNum] {
			continue
		}
		seen[tblNum] = true
		costCenter, _ := dbf.FieldInt(rec, "COST_CENTR")
		result = append(result, Table{
			Store:       dbf.FieldString(rec, "STORE"),
			TableNumber: tblNum,
			CostCenter:  costCenter,
		})
	}
	return result, nil
}
