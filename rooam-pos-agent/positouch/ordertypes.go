// ordertypes.go reads menu/order type definitions from MENUS.DBF.
package positouch

import (
	"fmt"
	"os"

	"rooam-pos-agent/dbf"
)

// OrderType represents a single POSitouch menu/order type.
type OrderType struct {
	MenuNumber  int    `json:"menu_number"`
	MenuTitle   string `json:"menu_title"`
	ContMenu    int    `json:"cont_menu"`
	FFOrderType int    `json:"ff_order_type"`
}

// ReadOrderTypes reads order/menu types from MENUS.DBF in the given SC directory.
func ReadOrderTypes(scPath string) ([]OrderType, error) {
	path := scPath + "MENUS.DBF"
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("positouch: MENUS.DBF not found in %s", scPath)
	}

	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read MENUS.DBF: %w", err)
	}

	orderTypes := make([]OrderType, 0, len(records))
	for _, rec := range records {
		ot := OrderType{
			MenuNumber:  int(floatField(rec, "menu_num")),
			MenuTitle:   stringField(rec, "menu_title"),
			ContMenu:    int(floatField(rec, "cont_menu")),
			FFOrderType: int(floatField(rec, "ff_ord_t")),
		}
		orderTypes = append(orderTypes, ot)
	}
	return orderTypes, nil
}
