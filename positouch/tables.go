package positouch

import (
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Table represents a unique table number extracted from CHECK.DBF.
type Table struct {
	TableNumber int `json:"table_number"`
}

// ReadTables reads CHECK.DBF from the given DBF directory, extracts all unique
// TABLE_NO values, and returns them as a sorted slice of Table records.
func ReadTables(dbfDir string) ([]Table, error) {
	path := filepath.Join(dbfDir, "CHECK.DBF")
	records, err := dbf.ReadFile(path)
	if err != nil {
		log.Printf("positouch: warning: cannot read CHECK.DBF (%s): %v", path, err)
		return []Table{}, nil
	}

	seen := make(map[int]struct{})
	for _, r := range records {
		n := intField(r, "TABLE_NO")
		if n > 0 {
			seen[n] = struct{}{}
		}
	}

	out := make([]Table, 0, len(seen))
	for n := range seen {
		out = append(out, Table{TableNumber: n})
	}
	log.Printf("positouch: found %d unique tables in %s", len(out), path)
	return out, nil
}
