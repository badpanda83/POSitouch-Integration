// Package positouch provides readers for POSitouch DBF data files.
package positouch

import (
	"fmt"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// CostCenter represents a POSitouch cost center.
type CostCenter struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// ReadCostCenters reads NAMECC.DBF from dbfDir.
// Falls back to NAMES.DBF filtering on CODE prefix "CC" if NAMECC.DBF is absent.
func ReadCostCenters(dbfDir string) ([]CostCenter, error) {
	path := findDBF(dbfDir, "NAMECC.DBF")
	if path != "" {
		return readNameFile(path, "")
	}

	// Fallback: NAMES.DBF filtered by CODE prefix "CC"
	fallback := findDBF(dbfDir, "NAMES.DBF")
	if fallback == "" {
		return nil, fmt.Errorf("positouch: NAMECC.DBF (and fallback NAMES.DBF) not found in %s", dbfDir)
	}
	return readNameFile(fallback, "CC")
}

// readNameFile reads a NAME????.DBF file (STORE/CODE/NAME layout).
// If prefix is non-empty, only records whose CODE starts with prefix are included.
func readNameFile(path, codePrefix string) ([]CostCenter, error) {
	df, err := dbf.Open(path)
	if err != nil {
		return nil, err
	}
	var results []CostCenter
	for _, rec := range df.Records {
		codeVal, _ := rec["CODE"]
		if codePrefix != "" {
			// CODE is a string in the fallback NAMES.DBF
			codeStr, ok := codeVal.(string)
			if !ok || !strings.HasPrefix(strings.ToUpper(codeStr), strings.ToUpper(codePrefix)) {
				continue
			}
			// Parse the numeric portion after the prefix
			numStr := strings.TrimSpace(codeStr[len(codePrefix):])
			code := 0
			if _, err := fmt.Sscanf(numStr, "%d", &code); err != nil {
				continue
			}
			name, _ := rec["NAME"].(string)
			results = append(results, CostCenter{Code: code, Name: strings.TrimSpace(name)})
		} else {
			code := int(toFloat64(codeVal))
			name, _ := rec["NAME"].(string)
			results = append(results, CostCenter{Code: code, Name: strings.TrimSpace(name)})
		}
	}
	return results, nil
}

