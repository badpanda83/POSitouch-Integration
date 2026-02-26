// Reads table data from CHKHDR.DBF (or CHECK.DBF fallback).
package positouch

import (
	"log"
	"path/filepath"

	"rooam-pos-agent/cache"
	"rooam-pos-agent/dbf"
)

// ReadTables reads unique table numbers from the DBF directory.
// Primary source: CHKHDR.DBF  (produced by POSIDBFW)
// Fallback:       CHECK.DBF   (produced by CHKTODBF)
// Table number 0 is ignored.
func ReadTables(dbfPath string) []cache.Table {
	primary := filepath.Join(dbfPath, "CHKHDR.DBF")
	records, err := dbf.ReadFile(primary)
	if err == nil {
		return parseTablesFromChkHdr(records)
	}
	log.Printf("[warn] tables: cannot read %q (%v); trying CHECK.DBF fallback", primary, err)

	fallback := filepath.Join(dbfPath, "CHECK.DBF")
	records, err = dbf.ReadFile(fallback)
	if err != nil {
		log.Printf("[warn] tables: cannot read fallback %q: %v", fallback, err)
		return nil
	}
	return parseTablesFromCheck(records)
}

func parseTablesFromChkHdr(records []map[string]interface{}) []cache.Table {
	seen := make(map[int]int) // table number → cost center
	for _, r := range records {
		tbl := int(floatField(r, "TABLE"))
		if tbl == 0 {
			continue
		}
		cc := int(floatField(r, "COST_CENTR"))
		seen[tbl] = cc
	}
	return tableMapToSlice(seen)
}

func parseTablesFromCheck(records []map[string]interface{}) []cache.Table {
	seen := make(map[int]int)
	for _, r := range records {
		tbl := int(floatField(r, "TABLE_NO"))
		if tbl == 0 {
			continue
		}
		cc := int(floatField(r, "COSTCENTER"))
		seen[tbl] = cc
	}
	return tableMapToSlice(seen)
}

func tableMapToSlice(m map[int]int) []cache.Table {
	out := make([]cache.Table, 0, len(m))
	for num, cc := range m {
		out = append(out, cache.Table{Number: num, CostCenter: cc})
	}
	return out
}
