package positouch

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"rooam-pos-agent/config"
	"rooam-pos-agent/dbf"
)

// Tender represents a single payment-type record.
type Tender struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// ReadTenders reads tender data from NAMEPAY.DBF.  If that file is not present
// it falls back to NAMES.DBF filtered by records whose CODE begins with "PY".
// Both the DBF and ALTDBF directories are tried.
func ReadTenders(cfg *config.Config) ([]Tender, error) {
	for _, dir := range []string{cfg.DBFDir, cfg.ALTDBFDir} {
		path := dir + "NAMEPAY.DBF"
		records, err := dbf.ReadFile(path)
		if err == nil {
			return parseTenders(records), nil
		}
		log.Printf("tenders: NAMEPAY.DBF not found in %s, trying fallback", dir)
	}

	for _, dir := range []string{cfg.DBFDir, cfg.ALTDBFDir} {
		path := dir + "NAMES.DBF"
		records, err := dbf.ReadFile(path)
		if err == nil {
			return parseTendersFromNames(records), nil
		}
		log.Printf("tenders: NAMES.DBF not found in %s", dir)
	}

	return nil, fmt.Errorf("tenders: no suitable DBF file found")
}

func parseTenders(records []map[string]string) []Tender {
	out := make([]Tender, 0, len(records))
	for _, r := range records {
		code, err := strconv.Atoi(strings.TrimSpace(r["CODE"]))
		if err != nil {
			continue
		}
		out = append(out, Tender{
			Code: code,
			Name: strings.TrimSpace(r["NAME"]),
		})
	}
	return out
}

func parseTendersFromNames(records []map[string]string) []Tender {
	out := make([]Tender, 0)
	for _, r := range records {
		if !strings.HasPrefix(r["CODE"], "PY") {
			continue
		}
		numStr := strings.TrimPrefix(r["CODE"], "PY")
		code, err := strconv.Atoi(strings.TrimSpace(numStr))
		if err != nil {
			continue
		}
		out = append(out, Tender{
			Code: code,
			Name: strings.TrimSpace(r["NAME"]),
		})
	}
	return out
}
