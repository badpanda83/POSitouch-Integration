// tenders.go reads payment type definitions from NAMEPAY.DBF (or NAMES.DBF).
package positouch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Tender represents a single POSitouch payment type.
type Tender struct {
	Store string `json:"store"`
	Code  int    `json:"code"`
	Name  string `json:"name"`
}

// ReadTenders reads payment types from NAMEPAY.DBF in dbfDir.
// If NAMEPAY.DBF does not exist, it falls back to NAMES.DBF filtering rows
// whose CODE field starts with "PY".
func ReadTenders(dbfDir string) ([]Tender, error) {
	primary := filepath.Join(dbfDir, "NAMEPAY.DBF")
	if _, err := os.Stat(primary); err == nil {
		return readNamePay(primary)
	}
	log.Printf("[positouch] NAMEPAY.DBF not found, falling back to NAMES.DBF")
	return readNamesForTenders(filepath.Join(dbfDir, "NAMES.DBF"))
}

func readNamePay(path string) ([]Tender, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read NAMEPAY.DBF: %w", err)
	}
	tenders := make([]Tender, 0, len(records))
	for _, rec := range records {
		tenders = append(tenders, Tender{
			Store: stringField(rec, "STORE"),
			Code:  int(floatField(rec, "CODE")),
			Name:  stringField(rec, "NAME"),
		})
	}
	log.Printf("[positouch] read %d tender(s) from NAMEPAY.DBF", len(tenders))
	return tenders, nil
}

func readNamesForTenders(path string) ([]Tender, error) {
	records, err := dbf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("positouch: read NAMES.DBF for tenders: %w", err)
	}
	tenders := make([]Tender, 0)
	for _, rec := range records {
		code := stringField(rec, "CODE")
		if !strings.HasPrefix(strings.ToUpper(code), "PY") {
			continue
		}
		tenders = append(tenders, Tender{
			Store: stringField(rec, "STORE"),
			Code:  parseCodeSuffix(strings.TrimSpace(code[2:])),
			Name:  stringField(rec, "NAME"),
		})
	}
	log.Printf("[positouch] read %d tender(s) from NAMES.DBF", len(tenders))
	return tenders, nil
}
