package positouch

import (
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Tender represents a payment type (cash, credit card, etc.).
type Tender struct {
	Code  int64  `json:"code"`
	Name  string `json:"name"`
	Store string `json:"store"`
}

// LoadTenders reads tender records from NAMEPAY.DBF.
// Falls back to NAMES.DBF filtered by CODE prefix "PY" if NAMEPAY.DBF is not found.
func LoadTenders(dbfDir string) ([]Tender, error) {
	primary := filepath.Join(dbfDir, "NAMEPAY.DBF")
	r, err := dbf.Open(primary)
	if err == nil {
		return parseTenders(r.Records()), nil
	}

	// Fallback: NAMES.DBF with CODE prefix "PY".
	fallback := filepath.Join(dbfDir, "NAMES.DBF")
	r, err = dbf.Open(fallback)
	if err != nil {
		return nil, err
	}
	return parseTendersFromNames(r.Records()), nil
}

func parseTenders(records []dbf.Record) []Tender {
	out := make([]Tender, 0, len(records))
	for _, rec := range records {
		out = append(out, Tender{
			Code:  rec.GetInt("CODE"),
			Name:  rec.GetString("NAME"),
			Store: rec.GetString("STORE"),
		})
	}
	return out
}

func parseTendersFromNames(records []dbf.Record) []Tender {
	out := make([]Tender, 0)
	for _, rec := range records {
		code := rec.GetString("CODE")
		if len(code) >= 2 && code[:2] == "PY" {
			out = append(out, Tender{
				Code:  rec.GetInt("CODE"),
				Name:  rec.GetString("NAME"),
				Store: rec.GetString("STORE"),
			})
		}
	}
	return out
}
