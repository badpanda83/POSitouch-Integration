// ordertypes.go reads menu/order type definitions from MENUS.DBF in the SC directory.
package positouch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// OrderType represents a single POSitouch menu/order type.
type OrderType struct {
	Store      string `json:"store"`
	MenuNumber int    `json:"menu_number"`
	Title      string `json:"title"`
	FFOrderType int   `json:"order_type"`
}

// ReadOrderTypes reads order/menu types from MENUS.DBF in scDir.
// MENUS.DBF is produced by SCRTODBF.EXE and lives in the SC directory.
func ReadOrderTypes(scDir string) ([]OrderType, error) {
	path := filepath.Join(scDir, "MENUS.DBF")
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("positouch: MENUS.DBF not found in %s", scDir)
	}

	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read MENUS.DBF: %w", err)
	}

	orderTypes := make([]OrderType, 0, len(records))
	for _, rec := range records {
		orderTypes = append(orderTypes, OrderType{
			Store:       stringField(rec, "STORE"),
			MenuNumber:  int(floatField(rec, "MENU_NUM")),
			Title:       stringField(rec, "MENU_TITLE"),
			FFOrderType: int(floatField(rec, "FF_ORD_T")),
		})
	}
	log.Printf("[positouch] read %d order type(s) from MENUS.DBF", len(orderTypes))
	return orderTypes, nil
}
