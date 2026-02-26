package positouch

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// ReadTenders reads payment type records from NAMEPAY.DBF in dbfDir.
// If NAMEPAY.DBF does not exist, it falls back to NAMES.DBF filtering rows
// whose CODE field starts with "PY".
func ReadTenders(dbfDir string) ([]map[string]interface{}, error) {
	primary := filepath.Join(dbfDir, "NAMEPAY.DBF")
	records, err := dbf.ReadFile(primary)
	if err == nil {
		log.Printf("[positouch] read %d tender(s) from NAMEPAY.DBF", len(records))
		return records, nil
	}

	log.Printf("[positouch] NAMEPAY.DBF not found (%v), trying NAMES.DBF fallback", err)
	result, fallbackErr := readNamesFallback(dbfDir, "PY")
	if fallbackErr != nil {
		return nil, fmt.Errorf("positouch: tenders: %w", fallbackErr)
	}
	return result, nil
}
