package positouch

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// ReadTenders reads NAMEPAY.DBF from dbfDir (or altDbfDir as fallback).
// Falls back to NAMES.DBF with prefix "PY" if NAMEPAY.DBF is not found.
func ReadTenders(dbfDir, altDbfDir string) ([]cache.Tender, error) {
	path, err := findDBF(dbfDir, altDbfDir, "NAMEPAY.DBF")
	if err != nil {
		return readNamesDBFTenders(dbfDir, altDbfDir)
	}
	return parseTenders(path)
}

func parseTenders(path string) ([]cache.Tender, error) {
	f, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("tenders: opening %s: %w", path, err)
	}
	var results []cache.Tender
	for _, rec := range f.Records() {
		t := cache.Tender{
			ID:   dbf.GetInt(rec, "CODE"),
			Name: strings.TrimSpace(dbf.GetString(rec, "NAME")),
		}
		if t.Name != "" {
			results = append(results, t)
		}
	}
	log.Printf("tenders: read %d records from %s", len(results), filepath.Base(path))
	return results, nil
}

// readNamesDBFTenders reads NAMES.DBF filtering by the "PY" prefix.
func readNamesDBFTenders(dbfDir, altDbfDir string) ([]cache.Tender, error) {
	path, err := findDBF(dbfDir, altDbfDir, "NAMES.DBF")
	if err != nil {
		return nil, err
	}
	f, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("tenders (NAMES.DBF): opening %s: %w", path, err)
	}
	var results []cache.Tender
	for _, rec := range f.Records() {
		code := dbf.GetString(rec, "CODE")
		if len(code) < 2 || !strings.EqualFold(code[:2], "PY") {
			continue
		}
		numStr := strings.TrimSpace(code[2:])
		var id int
		if _, err := fmt.Sscanf(numStr, "%d", &id); err != nil {
			log.Printf("tenders (NAMES.DBF): skipping entry with unparseable code %q: %v", code, err)
			continue
		}
		t := cache.Tender{
			ID:   id,
			Name: strings.TrimSpace(dbf.GetString(rec, "NAME")),
		}
		if t.Name != "" {
			results = append(results, t)
		}
	}
	log.Printf("tenders (NAMES.DBF/PY): read %d records", len(results))
	return results, nil
}
