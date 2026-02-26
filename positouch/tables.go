// tables.go reads unique table numbers from CHKHDR.DBF (or CHECK.DBF fallback).
package positouch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Table represents a unique table found in check header data.
type Table struct {
	Store      string `json:"store"`
	Number     int    `json:"number"`
	CostCenter int    `json:"cost_center"`
}

// ReadTables reads unique tables from CHKHDR.DBF in dbfDir.
// If CHKHDR.DBF does not exist, it falls back to CHECK.DBF.
func ReadTables(dbfDir string) ([]Table, error) {
	primary := filepath.Join(dbfDir, "CHKHDR.DBF")
	if _, err := os.Stat(primary); err == nil {
		return readChkHdr(primary)
	}
	log.Printf("[positouch] CHKHDR.DBF not found, falling back to CHECK.DBF")

	fallback := filepath.Join(dbfDir, "CHECK.DBF")
	if _, err := os.Stat(fallback); err != nil {
		return nil, fmt.Errorf("positouch: neither CHKHDR.DBF nor CHECK.DBF found in %s", dbfDir)
	}
	return readCheck(fallback)
}

func readChkHdr(path string) ([]Table, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read CHKHDR.DBF: %w", err)
	}
	tables := deduplicateTables(records, "TABLE", "COST_CENTR")
	log.Printf("[positouch] read %d unique table(s) from CHKHDR.DBF", len(tables))
	return tables, nil
}

func readCheck(path string) ([]Table, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read CHECK.DBF: %w", err)
	}
	tables := deduplicateTables(records, "TABLE_NO", "COSTCENTER")
	log.Printf("[positouch] read %d unique table(s) from CHECK.DBF", len(tables))
	return tables, nil
}

// deduplicateTables extracts unique table numbers from DBF records.
func deduplicateTables(records []map[string]interface{}, tableField, ccField string) []Table {
	seen := make(map[int]bool)
	tables := make([]Table, 0)
	for _, rec := range records {
		num := int(floatField(rec, tableField))
		if num == 0 || seen[num] {
			continue
		}
		seen[num] = true
		tables = append(tables, Table{
			Store:      stringField(rec, "STORE"),
			Number:     num,
			CostCenter: int(floatField(rec, ccField)),
		})
	}
	return tables
}
