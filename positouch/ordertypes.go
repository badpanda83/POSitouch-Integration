// ordertypes.go reads order type definitions from NAMECC.DBF in the DBF directory.
// In POSitouch, revenue centers (cost centers) double as order types.
package positouch

import (
"fmt"
"log"
"path/filepath"

"github.com/badpanda83/POSitouch-Integration/dbf"
)

// OrderType represents a single POSitouch order/revenue type.
type OrderType struct {
ID    int    `json:"id"`
Name  string `json:"name"`
}

// ReadOrderTypes reads order types from NAMECC.DBF in dbfDir.
// POSitouch uses revenue centers (cost centers) as order types.
func ReadOrderTypes(dbfDir string) ([]OrderType, error) {
path := filepath.Join(dbfDir, "NAMECC.DBF")
records, err := dbf.ReadFile(path)
if err != nil {
return nil, fmt.Errorf("positouch: read NAMECC.DBF: %w", err)
}

var orderTypes []OrderType
for _, rec := range records {
name := stringField(rec, "NAME")
code := int(floatField(rec, "CODE"))
if name == "" {
continue
}
orderTypes = append(orderTypes, OrderType{
ID:   code,
Name: name,
})
}
log.Printf("[positouch] read %d order type(s) from NAMECC.DBF", len(orderTypes))
return orderTypes, nil
}
