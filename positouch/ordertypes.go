package positouch

import (
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// OrderType represents a menu/order type from MENUS.DBF.
type OrderType struct {
	MenuNumber int64  `json:"menu_number"`
	Title      string `json:"title"`
	OrderType  int64  `json:"order_type"`
	Store      string `json:"store"`
}

// LoadOrderTypes reads order type records from MENUS.DBF (produced by SCRTODBF.EXE).
func LoadOrderTypes(dbfDir string) ([]OrderType, error) {
	path := filepath.Join(dbfDir, "MENUS.DBF")
	r, err := dbf.Open(path)
	if err != nil {
		return nil, err
	}
	return parseOrderTypes(r.Records()), nil
}

func parseOrderTypes(records []dbf.Record) []OrderType {
	out := make([]OrderType, 0, len(records))
	for _, rec := range records {
		out = append(out, OrderType{
			MenuNumber: rec.GetInt("MENU_NUM"),
			Title:      rec.GetString("MENU_TITLE"),
			OrderType:  rec.GetInt("FF_ORD_T"),
			Store:      rec.GetString("STORE"),
		})
	}
	return out
}
