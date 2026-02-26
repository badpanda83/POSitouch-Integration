package positouch

import (
	"fmt"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Tender represents a POSitouch payment type.
type Tender struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// ReadTenders reads NAMEPAY.DBF from dbfDir.
// Falls back to NAMES.DBF filtering on CODE prefix "PY" if NAMEPAY.DBF is absent.
func ReadTenders(dbfDir string) ([]Tender, error) {
	path := findDBF(dbfDir, "NAMEPAY.DBF")
	if path != "" {
		return readPayFile(path, "")
	}

	// Fallback: NAMES.DBF filtered by CODE prefix "PY"
	fallback := findDBF(dbfDir, "NAMES.DBF")
	if fallback == "" {
		return nil, fmt.Errorf("positouch: NAMEPAY.DBF (and fallback NAMES.DBF) not found in %s", dbfDir)
	}
	return readPayFile(fallback, "PY")
}

func readPayFile(path, codePrefix string) ([]Tender, error) {
	df, err := dbf.Open(path)
	if err != nil {
		return nil, err
	}
	var results []Tender
	for _, rec := range df.Records {
		codeVal, _ := rec["CODE"]
		if codePrefix != "" {
			codeStr, ok := codeVal.(string)
			if !ok || !strings.HasPrefix(strings.ToUpper(codeStr), strings.ToUpper(codePrefix)) {
				continue
			}
			numStr := strings.TrimSpace(codeStr[len(codePrefix):])
			code := 0
			if _, err := fmt.Sscanf(numStr, "%d", &code); err != nil {
				continue
			}
			name, _ := rec["NAME"].(string)
			results = append(results, Tender{Code: code, Name: strings.TrimSpace(name)})
		} else {
			code := int(toFloat64(codeVal))
			name, _ := rec["NAME"].(string)
			results = append(results, Tender{Code: code, Name: strings.TrimSpace(name)})
		}
	}
	return results, nil
}
