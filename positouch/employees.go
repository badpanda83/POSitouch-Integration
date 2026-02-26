package positouch

import (
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Employee represents a POSitouch employee/user record.
type Employee struct {
	UserNumber int    `json:"user_number"`
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	Type       int    `json:"type"`
	MagCardID  int    `json:"mag_card_id"`
	Store      string `json:"store"`
}

// ReadEmployees reads USERS.DBF from the given DBF directory and returns a
// slice of Employee records.  If EMPFILE.DBF also exists in the SC directory
// (scDir) it is used to enrich records with additional detail; if not found
// it is silently skipped.
func ReadEmployees(dbfDir, scDir string) ([]Employee, error) {
	path := filepath.Join(dbfDir, "USERS.DBF")
	records, err := dbf.ReadFile(path)
	if err != nil {
		log.Printf("positouch: warning: cannot read USERS.DBF (%s): %v", path, err)
		return []Employee{}, nil
	}

	out := make([]Employee, 0, len(records))
	for _, r := range records {
		e := Employee{
			Store:      stringField(r, "STORE"),
			UserNumber: intField(r, "USER_NBR"),
			LastName:   stringField(r, "NAME_LAST"),
			FirstName:  stringField(r, "NAME_FIRST"),
			Type:       intField(r, "TYPE"),
			MagCardID:  intField(r, "MAGCARD_ID"),
		}
		out = append(out, e)
	}
	log.Printf("positouch: read %d employees from %s", len(out), path)

	// Optionally enrich from EMPFILE.DBF (placed by TAW EXPORT in the SC dir).
	empfilePath := filepath.Join(scDir, "EMPFILE.DBF")
	empRecords, err := dbf.ReadFile(empfilePath)
	if err != nil {
		// EMPFILE.DBF is optional — log at debug level only.
		log.Printf("positouch: info: EMPFILE.DBF not available (%s): %v", empfilePath, err)
		return out, nil
	}

	// Build a lookup map by EMP_NUMBER so we can enrich the existing slice.
	empMap := make(map[int]map[string]interface{}, len(empRecords))
	for _, r := range empRecords {
		num := intField(r, "EMP_NUMBER")
		empMap[num] = r
	}

	for i, e := range out {
		if detail, ok := empMap[e.UserNumber]; ok {
			if ln := stringField(detail, "LAST_NAME"); ln != "" {
				out[i].LastName = ln
			}
			if fn := stringField(detail, "FIRST_NAME"); fn != "" {
				out[i].FirstName = fn
			}
		}
	}
	log.Printf("positouch: enriched employees with %d EMPFILE.DBF records from %s", len(empRecords), empfilePath)
	return out, nil
}
