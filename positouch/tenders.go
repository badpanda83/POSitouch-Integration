package positouch

import (
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Tender represents a POSitouch payment type (from NAMEPAY.DBF).
type Tender struct {
	Code  int    `json:"code"`
	Name  string `json:"name"`
	Store string `json:"store"`
}

// ReadTenders reads NAMEPAY.DBF from the given DBF directory and returns a
// slice of Tender records.
func ReadTenders(dbfDir string) ([]Tender, error) {
	path := filepath.Join(dbfDir, "NAMEPAY.DBF")
	records, err := dbf.ReadFile(path)
	if err != nil {
		log.Printf("positouch: warning: cannot read NAMEPAY.DBF (%s): %v", path, err)
		return []Tender{}, nil
	}

	out := make([]Tender, 0, len(records))
	for _, r := range records {
		t := Tender{
			Store: stringField(r, "STORE"),
			Code:  intField(r, "CODE"),
			Name:  stringField(r, "NAME"),
		}
		out = append(out, t)
	}
	log.Printf("positouch: read %d tenders from %s", len(out), path)
	return out, nil
}
