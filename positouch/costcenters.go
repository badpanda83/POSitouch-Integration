// Package positouch reads various POSitouch DBF files and returns structured data.
package positouch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// ReadCostCenters reads NAMECC.DBF from dbfDir (or altDbfDir as fallback).
// Falls back to NAMES.DBF with prefix "CC" if NAMECC.DBF is not found.
func ReadCostCenters(dbfDir, altDbfDir string) ([]cache.CostCenter, error) {
	path, err := findDBF(dbfDir, altDbfDir, "NAMECC.DBF")
	if err != nil {
		// Try fallback NAMES.DBF
		return readNamesDBF(dbfDir, altDbfDir, "CC")
	}
	return parseCostCenters(path)
}

func parseCostCenters(path string) ([]cache.CostCenter, error) {
	f, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("costcenters: opening %s: %w", path, err)
	}
	var results []cache.CostCenter
	for _, rec := range f.Records() {
		cc := cache.CostCenter{
			ID:   dbf.GetInt(rec, "CODE"),
			Name: strings.TrimSpace(dbf.GetString(rec, "NAME")),
		}
		if cc.Name != "" {
			results = append(results, cc)
		}
	}
	log.Printf("costcenters: read %d records from %s", len(results), filepath.Base(path))
	return results, nil
}

// readNamesDBF reads NAMES.DBF and filters by the given 2-letter prefix (e.g. "CC", "PY").
func readNamesDBF(dbfDir, altDbfDir, prefix string) ([]cache.CostCenter, error) {
	path, err := findDBF(dbfDir, altDbfDir, "NAMES.DBF")
	if err != nil {
		return nil, err
	}
	f, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("names fallback: opening %s: %w", path, err)
	}
	var results []cache.CostCenter
	for _, rec := range f.Records() {
		code := dbf.GetString(rec, "CODE")
		if len(code) < 2 || !strings.EqualFold(code[:2], prefix) {
			continue
		}
		// Parse the numeric portion (last 3 chars) as the ID
		numStr := strings.TrimSpace(code[2:])
		var id int
		if _, err := fmt.Sscanf(numStr, "%d", &id); err != nil {
			log.Printf("costcenters (NAMES.DBF): skipping entry with unparseable code %q: %v", code, err)
			continue
		}
		cc := cache.CostCenter{
			ID:   id,
			Name: strings.TrimSpace(dbf.GetString(rec, "NAME")),
		}
		if cc.Name != "" {
			results = append(results, cc)
		}
	}
	log.Printf("costcenters (NAMES.DBF/%s): read %d records", prefix, len(results))
	return results, nil
}

// findDBF looks for filename in dbfDir, falling back to altDbfDir.
func findDBF(dbfDir, altDbfDir, filename string) (string, error) {
	for _, dir := range []string{dbfDir, altDbfDir} {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, filename)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("dbf file not found: %s (checked %s and %s)", filename, dbfDir, altDbfDir)
}
