// employees.go reads employee data from USERS.DBF with optional enrichment
// from EMPFILE.DBF. SSN/SOC_SEC fields are intentionally never read.
package positouch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Employee represents a single POSitouch employee.
type Employee struct {
	Store     string `json:"store"`
	Number    int    `json:"number"`
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	Type      int    `json:"type"`
	MagCardID int    `json:"mag_card_id"`
	Status    string `json:"status,omitempty"` // from EMPFILE if available
}

// ReadEmployees reads employees from USERS.DBF in dbfDir and optionally
// enriches the records with status data from EMPFILE.DBF in scDir.
// SSN fields are intentionally never read or stored for security.
func ReadEmployees(dbfDir, scDir string) ([]Employee, error) {
	usersPath := filepath.Join(dbfDir, "USERS.DBF")
	records, err := dbf.ReadFile(usersPath)
	if err != nil {
		return nil, fmt.Errorf("positouch: read USERS.DBF: %w", err)
	}
	log.Printf("[positouch] read %d user(s) from USERS.DBF", len(records))

	employees := make([]Employee, 0, len(records))
	for _, rec := range records {
		employees = append(employees, Employee{
			Store:     stringField(rec, "STORE"),
			Number:    int(floatField(rec, "USER_NBR")),
			LastName:  stringField(rec, "NAME_LAST"),
			FirstName: stringField(rec, "NAME_FIRST"),
			Type:      int(floatField(rec, "TYPE")),
			MagCardID: int(floatField(rec, "MAGCARD_ID")),
		})
	}

	// Optionally enrich from EMPFILE.DBF (try scDir first, then dbfDir).
	empPath := filepath.Join(scDir, "EMPFILE.DBF")
	if _, err := os.Stat(empPath); err != nil {
		empPath = filepath.Join(dbfDir, "EMPFILE.DBF")
	}
	empRecords, empErr := dbf.ReadFile(empPath)
	if empErr != nil {
		log.Printf("[positouch] EMPFILE.DBF not found (%v), using USERS.DBF only", empErr)
		return employees, nil
	}
	log.Printf("[positouch] read %d record(s) from EMPFILE.DBF", len(empRecords))

	// Build a lookup from EMP_NUMBER → status, intentionally skipping SOC_SEC.
	statusByNum := make(map[int]string, len(empRecords))
	for _, rec := range empRecords {
		empNum := int(floatField(rec, "EMP_NUMBER"))
		// EMP_STATUS: F=Active, I=Inactive — never read SOC_SEC
		statusByNum[empNum] = stringField(rec, "EMP_STATUS")
	}

	for i := range employees {
		if status, ok := statusByNum[employees[i].Number]; ok {
			employees[i].Status = status
		}
	}
	return employees, nil
}
