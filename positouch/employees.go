package positouch

import (
	"fmt"

	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// Employee represents a POSitouch system user / employee.
type Employee struct {
	Number    int    `json:"number"`
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	Type      int    `json:"type"`
	MagCardID int    `json:"mag_card_id"`
	// Optional fields from EMPFILE.DBF
	Status   string `json:"status,omitempty"`   // F=Active, I=Inactive
	Phone    string `json:"phone,omitempty"`
	HireDate string `json:"hire_date,omitempty"`
}

// ReadEmployees reads USERS.DBF from dbfDir and optionally merges EMPFILE.DBF from scDir.
// SSN fields are never read or stored.
func ReadEmployees(dbfDir string, scDir string) ([]Employee, error) {
	usersPath := findDBF(dbfDir, "USERS.DBF")
	if usersPath == "" {
		return nil, fmt.Errorf("positouch: USERS.DBF not found in %s", dbfDir)
	}

	df, err := dbf.Open(usersPath)
	if err != nil {
		return nil, err
	}

	employees := make([]Employee, 0, len(df.Records))
	empByNumber := make(map[int]*Employee)

	for _, rec := range df.Records {
		emp := Employee{
			Number:    int(toFloat64(rec["USER_NBR"])),
			LastName:  toString(rec["NAME_LAST"]),
			FirstName: toString(rec["NAME_FIRST"]),
			Type:      int(toFloat64(rec["TYPE"])),
			MagCardID: int(toFloat64(rec["MAGCARD_ID"])),
		}
		employees = append(employees, emp)
		empByNumber[emp.Number] = &employees[len(employees)-1]
	}

	// Optionally merge EMPFILE.DBF — do not read SSN fields
	empFilePath := findDBF(scDir, "EMPFILE.DBF")
	if empFilePath != "" {
		empDF, err := dbf.Open(empFilePath)
		if err == nil {
			for _, rec := range empDF.Records {
				number := int(toFloat64(rec["EMP_NUMBER"]))
				emp, ok := empByNumber[number]
				if !ok {
					continue
				}
				emp.Status = toString(rec["EMP_STATUS"])
				emp.Phone = toString(rec["PHONE"])
				emp.HireDate = toString(rec["DATE_HIRED"])
			}
		}
	}

	return employees, nil
}
