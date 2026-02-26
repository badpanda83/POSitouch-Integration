// Package positouch provides typed readers for POSitouch DBF data files.
// costcenters.go reads cost center definitions from NAMECC.DBF (or NAMES.DBF).
package positouch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// CostCenter represents a single POSitouch cost center.
type CostCenter struct {
	Store string `json:"store"`
	Code  int    `json:"code"`
	Name  string `json:"name"`
}

// ReadCostCenters reads cost centers from NAMECC.DBF in dbfDir.
// If NAMECC.DBF does not exist, it falls back to NAMES.DBF filtering rows
// whose CODE field starts with "CC".
func ReadCostCenters(dbfDir string) ([]CostCenter, error) {
	primary := filepath.Join(dbfDir, "NAMECC.DBF")
	if _, err := os.Stat(primary); err == nil {
		return readNameCC(primary)
	}
	log.Printf("[positouch] NAMECC.DBF not found, falling back to NAMES.DBF")
	return readNamesForCostCenters(filepath.Join(dbfDir, "NAMES.DBF"))
}

func readNameCC(path string) ([]CostCenter, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read NAMECC.DBF: %w", err)
	}
	centers := make([]CostCenter, 0, len(records))
	for _, rec := range records {
		centers = append(centers, CostCenter{
			Store: stringField(rec, "STORE"),
			Code:  int(floatField(rec, "CODE")),
			Name:  stringField(rec, "NAME"),
		})
	}
	log.Printf("[positouch] read %d cost center(s) from NAMECC.DBF", len(centers))
	return centers, nil
}

func readNamesForCostCenters(path string) ([]CostCenter, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read NAMES.DBF for cost centers: %w", err)
	}
	centers := make([]CostCenter, 0)
	for _, rec := range records {
		code := stringField(rec, "CODE")
		if !strings.HasPrefix(strings.ToUpper(code), "CC") {
			continue
		}
		centers = append(centers, CostCenter{
			Store: stringField(rec, "STORE"),
			Code:  parseCodeSuffix(strings.TrimSpace(code[2:])),
			Name:  stringField(rec, "NAME"),
		})
	}
	log.Printf("[positouch] read %d cost center(s) from NAMES.DBF", len(centers))
	return centers, nil
}
