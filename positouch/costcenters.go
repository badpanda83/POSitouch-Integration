package positouch

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// CostCenter represents a single cost center (dining area / bar / etc.).
type CostCenter struct {
	Code  int64  `json:"code"`
	Name  string `json:"name"`
	Store string `json:"store"`
}

// LoadCostCenters reads cost center records from NAMECC.DBF.
// Falls back to NAMES.DBF filtered by CODE prefix "CC" if NAMECC.DBF is not found.
func LoadCostCenters(dbfDir string) ([]CostCenter, error) {
	primary := filepath.Join(dbfDir, "NAMECC.DBF")
	r, err := dbf.Open(primary)
	if err == nil {
		return parseCostCenters(r.Records()), nil
	}

	// Fallback: NAMES.DBF with CODE prefix "CC".
	fallback := filepath.Join(dbfDir, "NAMES.DBF")
	r, err = dbf.Open(fallback)
	if err != nil {
		return nil, err
	}
	return parseCostCentersFromNames(r.Records()), nil
}

func parseCostCenters(records []dbf.Record) []CostCenter {
	out := make([]CostCenter, 0, len(records))
	for _, rec := range records {
		out = append(out, CostCenter{
			Code:  rec.GetInt("CODE"),
			Name:  rec.GetString("NAME"),
			Store: rec.GetString("STORE"),
		})
	}
	return out
}

func parseCostCentersFromNames(records []dbf.Record) []CostCenter {
	out := make([]CostCenter, 0)
	for _, rec := range records {
		code := rec.GetString("CODE")
		if len(code) < 2 || strings.ToUpper(code[:2]) != "CC" {
			continue
		}
		var num int64
		if len(code) > 2 {
			n, err := strconv.ParseInt(strings.TrimSpace(code[2:]), 10, 64)
			if err == nil {
				num = n
			}
		}
		out = append(out, CostCenter{
			Code:  num,
			Name:  rec.GetString("NAME"),
			Store: rec.GetString("STORE"),
		})
	}
	return out
}
