// tables.go reads unique table numbers from CHKHDR.DBF (or CHECK.DBF).
package positouch

import (
	"fmt"
	"log"
	"os"

	"github.com/badpanda83/POSitouch-Integration/rooam-pos-agent/dbf"
)

// Table represents a unique table found in the check header file.
type Table struct {
	TableNumber int `json:"table_number"`
	CostCenter  int `json:"cost_center"`
}

// ReadTables reads unique tables from CHKHDR.DBF in the given DBF directory.
// If CHKHDR.DBF does not exist, it falls back to CHECK.DBF.
func ReadTables(dbfPath string) ([]Table, error) {
	primary := dbfPath + "CHKHDR.DBF"
	if _, err := os.Stat(primary); err == nil {
		return readChkHdr(primary)
	}
	log.Printf("positouch: CHKHDR.DBF not found, falling back to CHECK.DBF")

	fallback := dbfPath + "CHECK.DBF"
	if _, err := os.Stat(fallback); err != nil {
		return nil, fmt.Errorf("positouch: neither CHKHDR.DBF nor CHECK.DBF found in %s", dbfPath)
	}
	return readCheck(fallback)
}

func readChkHdr(path string) ([]Table, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read CHKHDR.DBF: %w", err)
	}
	return deduplicateTables(records, "TABLE", "COST_CENTR"), nil
}

func readCheck(path string) ([]Table, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read CHECK.DBF: %w", err)
	}
	return deduplicateTables(records, "TABLE_NO", "COSTCENTER"), nil
}

// deduplicateTables extracts unique table numbers from DBF records.
func deduplicateTables(records []map[string]interface{}, tableField, ccField string) []Table {
	seen := make(map[int]bool)
	tables := make([]Table, 0)
	for _, rec := range records {
		tableNum := int(floatField(rec, tableField))
		if tableNum == 0 {
			continue
		}
		if seen[tableNum] {
			continue
		}
		seen[tableNum] = true
		tables = append(tables, Table{
			TableNumber: tableNum,
			CostCenter:  int(floatField(rec, ccField)),
		})
	}
	return tables
}
