// tenders.go reads payment type definitions from NAMEPAY.DBF (or NAMES.DBF).
package positouch

import (
	"fmt"
	"log"
	"os"
	"strings"

	"rooam-pos-agent/dbf"
)

// Tender represents a single POSitouch payment type.
type Tender struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// ReadTenders reads payment types from NAMEPAY.DBF in the given DBF directory.
// If NAMEPAY.DBF does not exist, it falls back to NAMES.DBF filtering for
// records whose CODE field starts with "PY".
func ReadTenders(dbfPath string) ([]Tender, error) {
	primary := dbfPath + "NAMEPAY.DBF"
	if _, err := os.Stat(primary); err == nil {
		return readNamePay(primary)
	}
	log.Printf("positouch: NAMEPAY.DBF not found, falling back to NAMES.DBF")

	fallback := dbfPath + "NAMES.DBF"
	if _, err := os.Stat(fallback); err != nil {
		return nil, fmt.Errorf("positouch: neither NAMEPAY.DBF nor NAMES.DBF found in %s", dbfPath)
	}
	return readNamesForTenders(fallback)
}

func readNamePay(path string) ([]Tender, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read NAMEPAY.DBF: %w", err)
	}

	tenders := make([]Tender, 0, len(records))
	for _, rec := range records {
		code := int(floatField(rec, "CODE"))
		name := stringField(rec, "NAME")
		tenders = append(tenders, Tender{Code: code, Name: name})
	}
	return tenders, nil
}

func readNamesForTenders(path string) ([]Tender, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read NAMES.DBF: %w", err)
	}

	tenders := make([]Tender, 0)
	for _, rec := range records {
		code := stringField(rec, "CODE")
		if !strings.HasPrefix(strings.ToUpper(code), "PY") {
			continue
		}
		numStr := strings.TrimSpace(code[2:])
		num := parseCodeSuffix(numStr)
		name := stringField(rec, "NAME")
		tenders = append(tenders, Tender{Code: num, Name: name})
	}
	return tenders, nil
}
