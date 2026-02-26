// Reads order/menu type data from MENUS.DBF in the SC directory.
package positouch

import (
	"log"
	"path/filepath"

	"rooam-pos-agent/cache"
	"rooam-pos-agent/dbf"
)

// ReadOrderTypes reads menu/order type records from the SC directory.
// Source: MENUS.DBF (produced by SCRTODBF.EXE) — located in \SC\, NOT \DBF\.
func ReadOrderTypes(scPath string) []cache.OrderType {
	path := filepath.Join(scPath, "MENUS.DBF")
	records, err := dbf.ReadFile(path)
	if err != nil {
		log.Printf("[warn] ordertypes: cannot read %q: %v", path, err)
		return nil
	}
	return parseOrderTypes(records)
}

func parseOrderTypes(records []map[string]interface{}) []cache.OrderType {
	out := make([]cache.OrderType, 0, len(records))
	for _, r := range records {
		menuNum := int(floatField(r, "MENU_NUM"))
		title := stringField(r, "MENU_TITLE")
		contMenu := int(floatField(r, "CONT_MENU"))
		taxCode := int(floatField(r, "TAX_CODE"))
		ffOrdT := int(floatField(r, "FF_ORD_T"))

		if menuNum == 0 && title == "" {
			continue
		}

		out = append(out, cache.OrderType{
			MenuNumber:        menuNum,
			Title:             title,
			ContinuesOnMenu:   contMenu,
			TaxCode:           taxCode,
			FastFoodOrderType: ffOrdT,
		})
	}
	return out
}
