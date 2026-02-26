package positouch

import (
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// OrderType represents a POSitouch defined menu / order type (from MENUS.DBF).
// Store maps to the FF_ORD_T (fast-food order type) field in the DBF, which
// POSitouch uses as a store/area identifier for the menu.
type OrderType struct {
	MenuNumber          int    `json:"menu_number"`
	MenuTitle           string `json:"menu_title"`
	TaxCode             int    `json:"tax_code"`
	FastOrderCostCenter int    `json:"fast_order_cost_center"`
	Store               int    `json:"store"`
}

// ReadOrderTypes reads MENUS.DBF from the given DBF directory and returns a
// slice of OrderType records.
func ReadOrderTypes(dbfDir string) ([]OrderType, error) {
	path := filepath.Join(dbfDir, "MENUS.DBF")
	records, err := dbf.ReadFile(path)
	if err != nil {
		log.Printf("positouch: warning: cannot read MENUS.DBF (%s): %v", path, err)
		return []OrderType{}, nil
	}

	out := make([]OrderType, 0, len(records))
	for _, r := range records {
		ot := OrderType{
			MenuNumber:          intField(r, "MENU_NUM"),
			MenuTitle:           stringField(r, "MENU_TITLE"),
			TaxCode:             intField(r, "TAX_CODE"),
			FastOrderCostCenter: intField(r, "F_ORD_CC"),
			Store:               intField(r, "FF_ORD_T"),
		}
		out = append(out, ot)
	}
	log.Printf("positouch: read %d order types from %s", len(out), path)
	return out, nil
}
