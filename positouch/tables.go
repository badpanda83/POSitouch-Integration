package positouch

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// ReadTables reads unique table numbers from CHKHDR.DBF or CHECK.DBF.
// CHKHDR.DBF is tried first; if not found, CHECK.DBF is used.
func ReadTables(dbfDir, altDbfDir string) ([]cache.Table, error) {
	// Try CHKHDR.DBF first
	path, err := findDBF(dbfDir, altDbfDir, "CHKHDR.DBF")
	if err == nil {
		return parseChkhdr(path)
	}
	// Fall back to CHECK.DBF
	path, err = findDBF(dbfDir, altDbfDir, "CHECK.DBF")
	if err != nil {
		return nil, err
	}
	return parseCheck(path)
}

// parseChkhdr reads CHKHDR.DBF.
// Relevant fields: TABLE (field 6) and COST_CENTR (field 7).
func parseChkhdr(path string) ([]cache.Table, error) {
	f, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("tables: opening %s: %w", path, err)
	}
	seen := make(map[int]bool)
	var results []cache.Table
	for _, rec := range f.Records() {
		tableNo := dbf.GetInt(rec, "TABLE")
		if tableNo == 0 || seen[tableNo] {
			continue
		}
		seen[tableNo] = true
		results = append(results, cache.Table{
			TableNumber: tableNo,
			CostCenter:  dbf.GetInt(rec, "COST_CENTR"),
		})
	}
	log.Printf("tables: read %d unique tables from %s", len(results), filepath.Base(path))
	return results, nil
}

// parseCheck reads CHECK.DBF (produced by CHKTODBF).
// Relevant fields: TABLE_NO (field 6) and COSTCENTER (field 7).
func parseCheck(path string) ([]cache.Table, error) {
	f, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("tables: opening %s: %w", path, err)
	}
	seen := make(map[int]bool)
	var results []cache.Table
	for _, rec := range f.Records() {
		tableNo := dbf.GetInt(rec, "TABLE_NO")
		if tableNo == 0 || seen[tableNo] {
			continue
		}
		seen[tableNo] = true
		results = append(results, cache.Table{
			TableNumber: tableNo,
			CostCenter:  dbf.GetInt(rec, "COSTCENTER"),
		})
	}
	log.Printf("tables: read %d unique tables from %s", len(results), filepath.Base(path))
	return results, nil
}
