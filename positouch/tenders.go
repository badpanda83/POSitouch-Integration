// Reads tender / payment type data from NAMEPAY.DBF (or NAMES.DBF fallback).
package positouch

import (
	"log"
	"path/filepath"
	"strings"

	"rooam-pos-agent/cache"
	"rooam-pos-agent/dbf"
)

// ReadTenders reads tender records from the DBF directory.
// Primary source: NAMEPAY.DBF
// Fallback:       NAMES.DBF (records where CODE starts with "PY")
func ReadTenders(dbfPath string) []cache.Tender {
	primary := filepath.Join(dbfPath, "NAMEPAY.DBF")
	records, err := dbf.ReadFile(primary)
	if err == nil {
		return parseTendersFromNamePay(records)
	}
	log.Printf("[warn] tenders: cannot read %q (%v); trying NAMES.DBF fallback", primary, err)

	fallback := filepath.Join(dbfPath, "NAMES.DBF")
	records, err = dbf.ReadFile(fallback)
	if err != nil {
		log.Printf("[warn] tenders: cannot read fallback %q: %v", fallback, err)
		return nil
	}
	return parseTendersFromNames(records)
}

func parseTendersFromNamePay(records []map[string]interface{}) []cache.Tender {
	out := make([]cache.Tender, 0, len(records))
	for _, r := range records {
		code := int(floatField(r, "CODE"))
		name := stringField(r, "NAME")
		if name == "" && code == 0 {
			continue
		}
		out = append(out, cache.Tender{Code: code, Name: name})
	}
	return out
}

func parseTendersFromNames(records []map[string]interface{}) []cache.Tender {
	var out []cache.Tender
	for _, r := range records {
		code := stringField(r, "CODE")
		if !strings.HasPrefix(strings.ToUpper(code), "PY") {
			continue
		}
		name := stringField(r, "NAME")
		numCode := parseCodeSuffix(code, 2)
		out = append(out, cache.Tender{Code: numCode, Name: name})
	}
	return out
}
