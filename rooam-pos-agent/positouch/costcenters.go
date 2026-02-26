// Package positouch provides readers for individual POSitouch DBF data files.
// costcenters.go reads cost center definitions from NAMECC.DBF (or NAMES.DBF).
package positouch

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/rooam-pos-agent/dbf"
)

// CostCenter represents a single POSitouch cost center.
type CostCenter struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// ReadCostCenters reads cost centers from NAMECC.DBF in the given DBF directory.
// If NAMECC.DBF does not exist, it falls back to NAMES.DBF filtering for
// records whose CODE field starts with "CC".
func ReadCostCenters(dbfPath string) ([]CostCenter, error) {
	primary := dbfPath + "NAMECC.DBF"
	if _, err := os.Stat(primary); err == nil {
		return readNameCC(primary)
	}
	log.Printf("positouch: NAMECC.DBF not found, falling back to NAMES.DBF")

	fallback := dbfPath + "NAMES.DBF"
	if _, err := os.Stat(fallback); err != nil {
		return nil, fmt.Errorf("positouch: neither NAMECC.DBF nor NAMES.DBF found in %s", dbfPath)
	}
	return readNamesForCostCenters(fallback)
}

func readNameCC(path string) ([]CostCenter, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read NAMECC.DBF: %w", err)
	}

	centers := make([]CostCenter, 0, len(records))
	for _, rec := range records {
		code := int(floatField(rec, "CODE"))
		name := stringField(rec, "NAME")
		centers = append(centers, CostCenter{Code: code, Name: name})
	}
	return centers, nil
}

func readNamesForCostCenters(path string) ([]CostCenter, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read NAMES.DBF: %w", err)
	}

	centers := make([]CostCenter, 0)
	for _, rec := range records {
		code := stringField(rec, "CODE")
		if !strings.HasPrefix(strings.ToUpper(code), "CC") {
			continue
		}
		// Parse numeric portion after "CC"
		numStr := strings.TrimSpace(code[2:])
		num := parseCodeSuffix(numStr)
		name := stringField(rec, "NAME")
		centers = append(centers, CostCenter{Code: num, Name: name})
	}
	return centers, nil
}
