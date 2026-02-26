package positouch

import (
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// ReadOrderTypes reads menu/order-type records from MENUS.DBF in dbfDir.
func ReadOrderTypes(dbfDir string) ([]map[string]interface{}, error) {
	path := filepath.Join(dbfDir, "MENUS.DBF")
	records, err := dbf.ReadFile(path)
	if err != nil {
		log.Printf("[positouch] MENUS.DBF not found (%v), skipping order types", err)
		return nil, nil
	}
	log.Printf("[positouch] read %d order type(s) from MENUS.DBF", len(records))
	return records, nil
}
