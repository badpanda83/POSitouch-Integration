package positouch

import (
	"fmt"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// OrderType represents one row from MENUS.DBF.
type OrderType struct {
	Store      string `json:"store"`
	MenuNumber int    `json:"menu_number"`
	Title      string `json:"title"`
	OrderType  int    `json:"order_type"`
}

// ReadOrderTypes opens MENUS.DBF from dbfDir and returns all order types.
func ReadOrderTypes(dbfDir string) ([]OrderType, error) {
	path := filepath.Join(dbfDir, "MENUS.DBF")
	table, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("order types: %w", err)
	}

	result := make([]OrderType, 0, len(table.Records))
	for _, rec := range table.Records {
		menuNum, _ := dbf.FieldInt(rec, "MENU_NUM")
		ordType, _ := dbf.FieldInt(rec, "FF_ORD_T")
		result = append(result, OrderType{
			Store:      dbf.FieldString(rec, "STORE"),
			MenuNumber: menuNum,
			Title:      dbf.FieldString(rec, "MENU_TITLE"),
			OrderType:  ordType,
		})
	}
	return result, nil
}
