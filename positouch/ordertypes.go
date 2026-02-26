package positouch

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"rooam-pos-agent/config"
	"rooam-pos-agent/dbf"
)

// OrderType represents a single menu/order-type record from MENUS.DBF.
type OrderType struct {
	MenuNum     int    `json:"menu_num"`
	Title       string `json:"title"`
	TaxCode     int    `json:"tax_code"`
	FastOrderCC int    `json:"fast_order_cc"`
	OrderType   int    `json:"order_type"`
}

// ReadOrderTypes reads order-type data from MENUS.DBF in the SC directory.
// MENUS.DBF is produced by SCRTODBF.EXE.
func ReadOrderTypes(cfg *config.Config) ([]OrderType, error) {
	path := cfg.SCDir + "MENUS.DBF"
	records, err := dbf.ReadFile(path)
	if err != nil {
		log.Printf("order types: MENUS.DBF not found in %s", cfg.SCDir)
		return nil, fmt.Errorf("order types: %w", err)
	}
	return parseOrderTypes(records), nil
}

func parseOrderTypes(records []map[string]string) []OrderType {
	out := make([]OrderType, 0, len(records))
	for _, r := range records {
		menuNum, err := strconv.Atoi(strings.TrimSpace(r["menu_num"]))
		if err != nil {
			continue
		}
		taxCode, _ := strconv.Atoi(strings.TrimSpace(r["tax_code"]))
		fastOrderCC, _ := strconv.Atoi(strings.TrimSpace(r["f_ord_cc"]))
		orderType, _ := strconv.Atoi(strings.TrimSpace(r["ff_ord_t"]))

		out = append(out, OrderType{
			MenuNum:     menuNum,
			Title:       strings.TrimSpace(r["menu_title"]),
			TaxCode:     taxCode,
			FastOrderCC: fastOrderCC,
			OrderType:   orderType,
		})
	}
	return out
}
