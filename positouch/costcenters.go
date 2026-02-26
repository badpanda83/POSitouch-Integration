// Package positouch contains readers for the various POSitouch DBF data files.
package positouch

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"rooam-pos-agent/config"
	"rooam-pos-agent/dbf"
)

// CostCenter represents a single cost-center record.
type CostCenter struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// ReadCostCenters reads cost-center data from NAMECC.DBF.  If that file is not
// present it falls back to NAMES.DBF filtered by records whose CODE begins with
// "CC".  Both the DBF and ALTDBF directories are tried.
func ReadCostCenters(cfg *config.Config) ([]CostCenter, error) {
	// Try NAMECC.DBF first.
	for _, dir := range []string{cfg.DBFDir, cfg.ALTDBFDir} {
		path := dir + "NAMECC.DBF"
		records, err := dbf.ReadFile(path)
		if err == nil {
			return parseCostCenters(records), nil
		}
		log.Printf("cost centers: NAMECC.DBF not found in %s, trying fallback", dir)
	}

	// Fallback: NAMES.DBF filtered by CODE prefix "CC".
	for _, dir := range []string{cfg.DBFDir, cfg.ALTDBFDir} {
		path := dir + "NAMES.DBF"
		records, err := dbf.ReadFile(path)
		if err == nil {
			return parseCostCentersFromNames(records), nil
		}
		log.Printf("cost centers: NAMES.DBF not found in %s", dir)
	}

	return nil, fmt.Errorf("cost centers: no suitable DBF file found")
}

func parseCostCenters(records []map[string]string) []CostCenter {
	out := make([]CostCenter, 0, len(records))
	for _, r := range records {
		code, err := strconv.Atoi(strings.TrimSpace(r["CODE"]))
		if err != nil {
			continue
		}
		out = append(out, CostCenter{
			Code: code,
			Name: strings.TrimSpace(r["NAME"]),
		})
	}
	return out
}

func parseCostCentersFromNames(records []map[string]string) []CostCenter {
	out := make([]CostCenter, 0)
	for _, r := range records {
		if !strings.HasPrefix(r["CODE"], "CC") {
			continue
		}
		numStr := strings.TrimPrefix(r["CODE"], "CC")
		code, err := strconv.Atoi(strings.TrimSpace(numStr))
		if err != nil {
			continue
		}
		out = append(out, CostCenter{
			Code: code,
			Name: strings.TrimSpace(r["NAME"]),
		})
	}
	return out
}
