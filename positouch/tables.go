package positouch

import (
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// ReadTables reads unique table records from CHECK.DBF in dbfDir.
// If CHECK.DBF is not found, it falls back to CHKHDR.DBF.
// Only unique table numbers are returned.
func ReadTables(dbfDir string) ([]map[string]interface{}, error) {
	primary := filepath.Join(dbfDir, "CHECK.DBF")
	records, err := dbf.ReadFile(primary)
	if err == nil {
		unique := deduplicateTables(records, "TABLE_NO")
		log.Printf("[positouch] read %d unique table(s) from CHECK.DBF", len(unique))
		return unique, nil
	}

	log.Printf("[positouch] CHECK.DBF not found (%v), trying CHKHDR.DBF fallback", err)
	fallback := filepath.Join(dbfDir, "CHKHDR.DBF")
	records, err = dbf.ReadFile(fallback)
	if err != nil {
		log.Printf("[positouch] CHKHDR.DBF not found (%v), skipping tables", err)
		return nil, nil
	}
	unique := deduplicateTables(records, "TABLE")
	log.Printf("[positouch] read %d unique table(s) from CHKHDR.DBF", len(unique))
	return unique, nil
}

// deduplicateTables returns one record per unique value of the named field.
func deduplicateTables(records []map[string]interface{}, tableField string) []map[string]interface{} {
	seen := make(map[float64]struct{})
	var result []map[string]interface{}
	for _, rec := range records {
		tbl, ok := rec[tableField].(float64)
		if !ok {
			continue
		}
		if _, exists := seen[tbl]; !exists {
			seen[tbl] = struct{}{}
			result = append(result, rec)
		}
	}
	return result
}
