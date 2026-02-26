package positouch

import (
	"fmt"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Table represents a unique POSitouch table number.
type Table struct {
	Number int `json:"number"`
}

// ReadTables reads unique table numbers from CHKHDR.DBF or CHECK.DBF in dbfDir.
// Tries CHKHDR.DBF first; falls back to CHECK.DBF.
func ReadTables(dbfDir string) ([]Table, error) {
	var path, tableField string

	if p := findDBF(dbfDir, "CHKHDR.DBF"); p != "" {
		path = p
		tableField = "TABLE"
	} else if p := findDBF(dbfDir, "CHECK.DBF"); p != "" {
		path = p
		tableField = "TABLE_NO"
	} else {
		return nil, fmt.Errorf("positouch: neither CHKHDR.DBF nor CHECK.DBF found in %s", dbfDir)
	}

	df, err := dbf.Open(path)
	if err != nil {
		return nil, err
	}

	seen := make(map[int]struct{})
	var tables []Table
	for _, rec := range df.Records {
		n := int(toFloat64(rec[tableField]))
		if n <= 0 {
			continue
		}
		if _, exists := seen[n]; !exists {
			seen[n] = struct{}{}
			tables = append(tables, Table{Number: n})
		}
	}
	return tables, nil
}
