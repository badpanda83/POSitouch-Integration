// Package positouch reads POSitouch DBF export files and returns typed models.
package positouch

import (
	"fmt"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// CostCenter represents one row from NAMECC.DBF.
type CostCenter struct {
	Store string `json:"store"`
	Code  int    `json:"code"`
	Name  string `json:"name"`
}

// ReadCostCenters opens NAMECC.DBF from dbfDir and returns all cost centers.
func ReadCostCenters(dbfDir string) ([]CostCenter, error) {
	path := filepath.Join(dbfDir, "NAMECC.DBF")
	table, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cost centers: %w", err)
	}

	result := make([]CostCenter, 0, len(table.Records))
	for _, rec := range table.Records {
		code, _ := dbf.FieldInt(rec, "CODE")
		result = append(result, CostCenter{
			Store: dbf.FieldString(rec, "STORE"),
			Code:  code,
			Name:  dbf.FieldString(rec, "NAME"),
		})
	}
	return result, nil
}
