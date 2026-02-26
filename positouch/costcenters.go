// Package positouch provides readers for individual POSitouch DBF data files.
// This file reads cost center data from NAMECC.DBF (or NAMES.DBF fallback).
package positouch

import (
	"log"
	"path/filepath"
	"strings"

	"rooam-pos-agent/cache"
	"rooam-pos-agent/dbf"
)

// ReadCostCenters reads cost center records from the DBF directory.
// Primary source: NAMECC.DBF
// Fallback:       NAMES.DBF (records where CODE starts with "CC")
func ReadCostCenters(dbfPath string) []cache.CostCenter {
	primary := filepath.Join(dbfPath, "NAMECC.DBF")
	records, err := dbf.ReadFile(primary)
	if err == nil {
		return parseCostCentersFromNameCC(records)
	}
	log.Printf("[warn] costcenters: cannot read %q (%v); trying NAMES.DBF fallback", primary, err)

	fallback := filepath.Join(dbfPath, "NAMES.DBF")
	records, err = dbf.ReadFile(fallback)
	if err != nil {
		log.Printf("[warn] costcenters: cannot read fallback %q: %v", fallback, err)
		return nil
	}
	return parseCostCentersFromNames(records)
}

func parseCostCentersFromNameCC(records []map[string]interface{}) []cache.CostCenter {
	out := make([]cache.CostCenter, 0, len(records))
	for _, r := range records {
		code := int(floatField(r, "CODE"))
		name := stringField(r, "NAME")
		if name == "" && code == 0 {
			continue
		}
		out = append(out, cache.CostCenter{Code: code, Name: name})
	}
	return out
}

func parseCostCentersFromNames(records []map[string]interface{}) []cache.CostCenter {
	var out []cache.CostCenter
	for _, r := range records {
		code := stringField(r, "CODE")
		if !strings.HasPrefix(strings.ToUpper(code), "CC") {
			continue
		}
		name := stringField(r, "NAME")
		numCode := parseCodeSuffix(code, 2)
		out = append(out, cache.CostCenter{Code: numCode, Name: name})
	}
	return out
}
