// employees.go reads employee data from USERS.DBF with optional enrichment
// from EMPFILE.DBF.  SSN/SOC_SEC fields are intentionally never read.
package positouch

import (
	"fmt"
	"log"
	"os"

	"github.com/badpanda83/POSitouch-Integration/rooam-pos-agent/dbf"
)

// Employee represents a single POSitouch employee.
type Employee struct {
	UserNumber int    `json:"user_number"`
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	Type       int    `json:"type"`
	MagCardID  int    `json:"mag_card_id"`
	// Extended fields from EMPFILE.DBF (optional)
	EmpType    string `json:"emp_type,omitempty"`
	EmpStatus  string `json:"emp_status,omitempty"`
	Phone      string `json:"phone,omitempty"`
	DateHired  string `json:"date_hired,omitempty"`
}

// ReadEmployees reads employees from USERS.DBF in the given DBF directory and
// optionally enriches the records with data from EMPFILE.DBF in the SC directory.
// SSN fields are intentionally skipped for security.
func ReadEmployees(dbfPath, scPath string) ([]Employee, error) {
	usersFile := dbfPath + "USERS.DBF"
	if _, err := os.Stat(usersFile); err != nil {
		return nil, fmt.Errorf("positouch: USERS.DBF not found in %s", dbfPath)
	}

	records, err := dbf.ReadFile(usersFile)
	if err != nil {
		return nil, fmt.Errorf("positouch: read USERS.DBF: %w", err)
	}

	employees := make([]Employee, 0, len(records))
	for _, rec := range records {
		emp := Employee{
			UserNumber: int(floatField(rec, "USER_NBR")),
			LastName:   stringField(rec, "NAME_LAST"),
			FirstName:  stringField(rec, "NAME_FIRST"),
			Type:       int(floatField(rec, "TYPE")),
			MagCardID:  int(floatField(rec, "MAGCARD_ID")),
		}
		employees = append(employees, emp)
	}

	// Optionally enrich from EMPFILE.DBF
	empFile := scPath + "EMPFILE.DBF"
	if _, err := os.Stat(empFile); err != nil {
		log.Printf("positouch: EMPFILE.DBF not found in %s, skipping enrichment", scPath)
		return employees, nil
	}

	empRecords, err := dbf.ReadFile(empFile)
	if err != nil {
		log.Printf("positouch: warning reading EMPFILE.DBF: %v", err)
		return employees, nil
	}

	// Build a lookup by mag card number (CARD_NUM ↔ MAGCARD_ID)
	type empExtended struct {
		EmpType   string
		EmpStatus string
		Phone     string
		DateHired string
	}
	extByCard := make(map[int]empExtended, len(empRecords))
	for _, rec := range empRecords {
		cardNum := int(floatField(rec, "CARD_NUM"))
		// Do NOT read SOC_SEC / SSN fields
		ext := empExtended{
			EmpType:   stringField(rec, "EMP_TYPE"),
			EmpStatus: stringField(rec, "EMP_STATUS"),
			Phone:     stringField(rec, "PHONE"),
			DateHired: stringField(rec, "DATE_HIRED"),
		}
		extByCard[cardNum] = ext
	}

	for i := range employees {
		if ext, ok := extByCard[employees[i].MagCardID]; ok {
			employees[i].EmpType = ext.EmpType
			employees[i].EmpStatus = ext.EmpStatus
			employees[i].Phone = ext.Phone
			employees[i].DateHired = ext.DateHired
		}
	}

	return employees, nil
}
