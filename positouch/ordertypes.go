package positouch

import (
	"fmt"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// OrderType represents a POSitouch menu / order type entry from MENUS.DBF.
type OrderType struct {
	MenuNum   int    `json:"menu_num"`
	Title     string `json:"title"`
	ContMenu  int    `json:"cont_menu"`
	OrderType int    `json:"order_type"` // FF_ORD_T field
}

// ReadOrderTypes reads MENUS.DBF from scDir (produced by SCRTODBF.EXE).
func ReadOrderTypes(scDir string) ([]OrderType, error) {
	path := findDBF(scDir, "MENUS.DBF")
	if path == "" {
		return nil, fmt.Errorf("positouch: MENUS.DBF not found in %s", scDir)
	}

	df, err := dbf.Open(path)
	if err != nil {
		return nil, err
	}

	results := make([]OrderType, 0, len(df.Records))
	for _, rec := range df.Records {
		ot := OrderType{
			MenuNum:   int(toFloat64(rec["MENU_NUM"])),
			Title:     toString(rec["MENU_TITLE"]),
			ContMenu:  int(toFloat64(rec["CONT_MENU"])),
			OrderType: int(toFloat64(rec["FF_ORD_T"])),
		}
		results = append(results, ot)
	}
	return results, nil
}
