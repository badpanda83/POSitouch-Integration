package positouch

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// ReadOrderTypes reads MENUS.DBF from the SC directory.
// MENUS.DBF is produced by SCRTODBF.EXE and lives in the SC folder, not the DBF folder.
func ReadOrderTypes(scDir string) ([]cache.OrderType, error) {
	path := filepath.Join(scDir, "MENUS.DBF")
	f, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ordertypes: opening %s: %w", path, err)
	}
	var results []cache.OrderType
	for _, rec := range f.Records() {
		ot := cache.OrderType{
			ID:    dbf.GetInt(rec, "menu_num"),
			Name:  strings.TrimSpace(dbf.GetString(rec, "menu_title")),
			FFOrd: dbf.GetInt(rec, "ff_ord_t"),
		}
		if ot.ID != 0 && ot.Name != "" {
			results = append(results, ot)
		}
	}
	log.Printf("ordertypes: read %d records from %s", len(results), filepath.Base(path))
	return results, nil
}
