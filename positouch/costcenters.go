// Package positouch provides readers for POSitouch DBF data files.
package positouch

import (
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// CostCenter represents a POSitouch cost center (from NAMECC.DBF).
type CostCenter struct {
	Code  int    `json:"code"`
	Name  string `json:"name"`
	Store string `json:"store"`
}

// ReadCostCenters reads NAMECC.DBF from the given DBF directory and returns
// a slice of CostCenter records.  Missing files are logged as warnings rather
// than returned as errors so the agent can continue with other data types.
func ReadCostCenters(dbfDir string) ([]CostCenter, error) {
	path := filepath.Join(dbfDir, "NAMECC.DBF")
	records, err := dbf.ReadFile(path)
	if err != nil {
		log.Printf("positouch: warning: cannot read NAMECC.DBF (%s): %v", path, err)
		return []CostCenter{}, nil
	}

	out := make([]CostCenter, 0, len(records))
	for _, r := range records {
		cc := CostCenter{
			Store: stringField(r, "STORE"),
			Code:  intField(r, "CODE"),
			Name:  stringField(r, "NAME"),
		}
		out = append(out, cc)
	}
	log.Printf("positouch: read %d cost centers from %s", len(out), path)
	return out, nil
}
