// Package positouch reads POSitouch DBF files and converts them to plain Go maps.
// This file handles cost center data from NAMECC.DBF (or NAMES.DBF fallback).
package positouch

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// ReadCostCenters reads cost center records from NAMECC.DBF in dbfDir.
// If NAMECC.DBF does not exist, it falls back to NAMES.DBF filtering rows
// whose CODE field starts with "CC".
func ReadCostCenters(dbfDir string) ([]map[string]interface{}, error) {
	primary := filepath.Join(dbfDir, "NAMECC.DBF")
	records, err := dbf.ReadFile(primary)
	if err == nil {
		log.Printf("[positouch] read %d cost center(s) from NAMECC.DBF", len(records))
		return records, nil
	}

	// Primary file missing — try case-insensitive glob then fall back to NAMES.DBF.
	log.Printf("[positouch] NAMECC.DBF not found (%v), trying NAMES.DBF fallback", err)
	return readNamesFallback(dbfDir, "CC")
}

// readNamesFallback reads NAMES.DBF and returns rows whose CODE field starts
// with the given prefix (e.g. "CC" for cost centers, "PY" for tenders).
func readNamesFallback(dbfDir, prefix string) ([]map[string]interface{}, error) {
	path := filepath.Join(dbfDir, "NAMES.DBF")
	all, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: NAMES.DBF fallback failed: %w", err)
	}

	var filtered []map[string]interface{}
	for _, row := range all {
		code, _ := row["CODE"].(string)
		if strings.HasPrefix(strings.ToUpper(code), strings.ToUpper(prefix)) {
			filtered = append(filtered, row)
		}
	}
	log.Printf("[positouch] read %d record(s) from NAMES.DBF with prefix %q", len(filtered), prefix)
	return filtered, nil
}
