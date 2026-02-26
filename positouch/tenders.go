package positouch

import (
	"fmt"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Tender represents one row from NAMEPAY.DBF.
// Code 0 = Cash; codes 1-19 are user-definable payment types.
type Tender struct {
	Store string `json:"store"`
	Code  int    `json:"code"`
	Name  string `json:"name"`
}

// ReadTenders opens NAMEPAY.DBF from dbfDir and returns all tenders.
func ReadTenders(dbfDir string) ([]Tender, error) {
	path := filepath.Join(dbfDir, "NAMEPAY.DBF")
	table, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("tenders: %w", err)
	}

	result := make([]Tender, 0, len(table.Records))
	for _, rec := range table.Records {
		code, _ := dbf.FieldInt(rec, "CODE")
		result = append(result, Tender{
			Store: dbf.FieldString(rec, "STORE"),
			Code:  code,
			Name:  dbf.FieldString(rec, "NAME"),
		})
	}
	return result, nil
}
