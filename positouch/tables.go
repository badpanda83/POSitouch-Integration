package positouch

import (
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Table represents a dining table extracted from check header data.
type Table struct {
	TableNumber int64  `json:"table_number"`
	CostCenter  int64  `json:"cost_center"`
	Store       string `json:"store"`
}

// LoadTables reads table records from CHKHDR.DBF (produced by CHKTODBF).
// Falls back to CHECK.DBF if CHKHDR.DBF is not found.
// Deduplicates by table number + cost center combination.
func LoadTables(dbfDir string) ([]Table, error) {
	primary := filepath.Join(dbfDir, "CHKHDR.DBF")
	r, err := dbf.Open(primary)
	if err == nil {
		return parseTables(r.Records()), nil
	}

	fallback := filepath.Join(dbfDir, "CHECK.DBF")
	r, err = dbf.Open(fallback)
	if err != nil {
		return nil, err
	}
	return parseTables(r.Records()), nil
}

func parseTables(records []dbf.Record) []Table {
	type key struct {
		table int64
		cc    int64
	}
	seen := make(map[key]bool)
	out := make([]Table, 0)

	for _, rec := range records {
		tableNum := rec.GetInt("TABLE")
		if tableNum == 0 {
			continue
		}
		cc := rec.GetInt("COST_CENTR")
		k := key{tableNum, cc}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, Table{
			TableNumber: tableNum,
			CostCenter:  cc,
			Store:       rec.GetString("STORE"),
		})
	}
	return out
}
