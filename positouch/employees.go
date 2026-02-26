package positouch

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"rooam-pos-agent/config"
	"rooam-pos-agent/dbf"
)

// Employee represents a single employee record.
type Employee struct {
	Number    int    `json:"number"`
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	Type      int    `json:"type"`
	MagCardID int    `json:"mag_card_id"`
	Status    string `json:"status"`
}

// ReadEmployees reads employee data from USERS.DBF.  If EMPFILE.DBF is also
// present in the SC directory, it is used to enrich status information.
func ReadEmployees(cfg *config.Config) ([]Employee, error) {
	var usersRecords []map[string]string
	var usersErr error

	for _, dir := range []string{cfg.DBFDir, cfg.ALTDBFDir} {
		path := dir + "USERS.DBF"
		usersRecords, usersErr = dbf.ReadFile(path)
		if usersErr == nil {
			break
		}
		log.Printf("employees: USERS.DBF not found in %s", dir)
	}

	if usersErr != nil {
		return nil, fmt.Errorf("employees: no suitable USERS.DBF found")
	}

	employees := parseEmployees(usersRecords)

	// Optional enrichment from EMPFILE.DBF in the SC directory.
	empPath := cfg.SCDir + "EMPFILE.DBF"
	empRecords, err := dbf.ReadFile(empPath)
	if err == nil {
		enrichEmployees(employees, empRecords)
	}

	return employees, nil
}

func parseEmployees(records []map[string]string) []Employee {
	out := make([]Employee, 0, len(records))
	for _, r := range records {
		num, err := strconv.Atoi(strings.TrimSpace(r["USER_NBR"]))
		if err != nil {
			continue
		}
		userType, _ := strconv.Atoi(strings.TrimSpace(r["TYPE"]))
		magCard, _ := strconv.Atoi(strings.TrimSpace(r["MAGCARD_ID"]))

		out = append(out, Employee{
			Number:    num,
			LastName:  r["NAME_LAST"],
			FirstName: r["NAME_FIRST"],
			Type:      userType,
			MagCardID: magCard,
		})
	}
	return out
}

// enrichEmployees merges EMPFILE data into the employee slice.
// SSN fields are intentionally skipped for security.
func enrichEmployees(employees []Employee, empRecords []map[string]string) {
	// Build lookup by employee number.
	byNumber := make(map[int]*Employee, len(employees))
	for i := range employees {
		byNumber[employees[i].Number] = &employees[i]
	}

	for _, r := range empRecords {
		num, err := strconv.Atoi(strings.TrimSpace(r["EMP_NUMBER"]))
		if err != nil {
			continue
		}
		emp, ok := byNumber[num]
		if !ok {
			continue
		}
		if status, exists := r["EMP_STATUS"]; exists {
			emp.Status = strings.TrimSpace(status)
		}
	}
}
