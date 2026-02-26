package positouch

import (
	"log"
	"path/filepath"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// socSecField is the EMPFILE.DBF field that must never be cached.
const socSecField = "SOC_SEC"

// ReadEmployees reads employee records from USERS.DBF and, if present,
// EMPFILE.DBF in dbfDir. Records from EMPFILE.DBF are merged by employee
// number, and the SOC_SEC field is always dropped for security.
func ReadEmployees(dbfDir string) ([]map[string]interface{}, error) {
	usersPath := filepath.Join(dbfDir, "USERS.DBF")
	users, err := dbf.ReadFile(usersPath)
	if err != nil {
		log.Printf("[positouch] USERS.DBF not found (%v), skipping employees", err)
		return nil, nil
	}
	log.Printf("[positouch] read %d user(s) from USERS.DBF", len(users))

	// Build a lookup from USER_NBR → USERS record for merging.
	userMap := make(map[float64]map[string]interface{}, len(users))
	for _, u := range users {
		if nbr, ok := u["USER_NBR"].(float64); ok {
			userMap[nbr] = u
		}
	}

	// Attempt to read the extended employee file.
	empPath := filepath.Join(dbfDir, "EMPFILE.DBF")
	empRecords, empErr := dbf.ReadFile(empPath)
	if empErr != nil {
		log.Printf("[positouch] EMPFILE.DBF not found (%v), using USERS.DBF only", empErr)
		return users, nil
	}
	log.Printf("[positouch] read %d record(s) from EMPFILE.DBF", len(empRecords))

	// Merge EMPFILE data into the USERS records, dropping SOC_SEC.
	for _, emp := range empRecords {
		// Drop the social security field before any processing.
		delete(emp, socSecField)

		empNum, ok := emp["EMP_NUMBER"].(float64)
		if !ok {
			continue
		}
		if existing, found := userMap[empNum]; found {
			for k, v := range emp {
				existing[k] = v
			}
		} else {
			userMap[empNum] = emp
		}
	}

	// Collect merged results.
	result := make([]map[string]interface{}, 0, len(userMap))
	for _, rec := range userMap {
		delete(rec, socSecField) // ensure it is absent after merge
		result = append(result, rec)
	}
	return result, nil
}
