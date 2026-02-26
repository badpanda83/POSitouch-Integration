// Reads employee data from USERS.DBF and optionally EMPFILE.DBF.
// SSN / Social Security data is never read or stored.
package positouch

import (
	"log"
	"path/filepath"

	"rooam-pos-agent/cache"
	"rooam-pos-agent/dbf"
)

// ReadEmployees reads employee records from the DBF and SC directories.
// Primary source:  USERS.DBF  (dbfPath)
// Optional extend: EMPFILE.DBF (scPath) produced by TAW EXPORT
func ReadEmployees(dbfPath, scPath string) []cache.Employee {
	primary := filepath.Join(dbfPath, "USERS.DBF")
	records, err := dbf.ReadFile(primary)
	if err != nil {
		log.Printf("[warn] employees: cannot read %q: %v", primary, err)
		return nil
	}

	employees := parseUsersDBF(records)

	// Build a lookup by user number for optional enrichment from EMPFILE.
	empByNum := make(map[int]*cache.Employee, len(employees))
	for i := range employees {
		empByNum[employees[i].UserNumber] = &employees[i]
	}

	empfilePath := filepath.Join(scPath, "EMPFILE.DBF")
	empRecords, err := dbf.ReadFile(empfilePath)
	if err != nil {
		// EMPFILE is optional — log at debug level and continue.
		log.Printf("[info] employees: EMPFILE.DBF not available at %q (%v); skipping extended data", empfilePath, err)
		return employees
	}

	mergeEmpFile(employees, empByNum, empRecords)
	return employees
}

func parseUsersDBF(records []map[string]interface{}) []cache.Employee {
	out := make([]cache.Employee, 0, len(records))
	for _, r := range records {
		userNum := int(floatField(r, "USER_NBR"))
		lastName := stringField(r, "NAME_LAST")
		firstName := stringField(r, "NAME_FIRST")
		empType := int(floatField(r, "TYPE"))
		magCard := int(floatField(r, "MAGCARD_ID"))

		if userNum == 0 && lastName == "" && firstName == "" {
			continue
		}

		out = append(out, cache.Employee{
			UserNumber: userNum,
			LastName:   lastName,
			FirstName:  firstName,
			Type:       empType,
			MagCardID:  magCard,
		})
	}
	return out
}

// mergeEmpFile enriches the employees slice with data from EMPFILE.DBF.
// NOTE: SSN fields are intentionally ignored and never stored.
func mergeEmpFile(employees []cache.Employee, empByNum map[int]*cache.Employee, records []map[string]interface{}) {
	for _, r := range records {
		empNum := int(floatField(r, "EMP_NUMBER"))
		if emp, ok := empByNum[empNum]; ok {
			emp.Status = stringField(r, "EMP_STATUS")
			emp.Phone = stringField(r, "PHONE")
			emp.DateHired = stringField(r, "DATE_HIRED")
			// SSN is deliberately not read or stored.
		}
	}
}
