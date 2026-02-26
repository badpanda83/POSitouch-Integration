package positouch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/dbf"
)

// ReadEmployees reads USERS.DBF and, if present, enriches data from EMPFILE.DBF.
// Both files are expected in scDir (the SC folder), not the DBF folder.
func ReadEmployees(dbfDir, altDbfDir, scDir string) ([]cache.Employee, error) {
	path, err := findDBF(dbfDir, altDbfDir, "USERS.DBF")
	if err != nil {
		return nil, err
	}
	employees, err := parseUsers(path)
	if err != nil {
		return nil, err
	}

	// Try to enrich from EMPFILE.DBF in the SC directory (optional).
	empfilePath := filepath.Join(scDir, "EMPFILE.DBF")
	if _, statErr := os.Stat(empfilePath); statErr == nil {
		enriched, enrichErr := enrichFromEmpfile(employees, empfilePath)
		if enrichErr != nil {
			log.Printf("employees: warning — could not read EMPFILE.DBF: %v", enrichErr)
		} else {
			employees = enriched
		}
	}

	return employees, nil
}

func parseUsers(path string) ([]cache.Employee, error) {
	f, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("employees: opening %s: %w", path, err)
	}
	var results []cache.Employee
	for _, rec := range f.Records() {
		e := cache.Employee{
			ID:        dbf.GetInt(rec, "USER_NBR"),
			LastName:  strings.TrimSpace(dbf.GetString(rec, "NAME_LAST")),
			FirstName: strings.TrimSpace(dbf.GetString(rec, "NAME_FIRST")),
			Type:      dbf.GetInt(rec, "TYPE"),
			MagCardID: dbf.GetInt(rec, "MAGCARD_ID"),
		}
		if e.ID != 0 {
			results = append(results, e)
		}
	}
	log.Printf("employees: read %d records from %s", len(results), filepath.Base(path))
	return results, nil
}

// enrichFromEmpfile merges additional employee data from EMPFILE.DBF.
// It matches on employee number and overwrites name fields when available.
func enrichFromEmpfile(employees []cache.Employee, path string) ([]cache.Employee, error) {
	f, err := dbf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("empfile: opening %s: %w", path, err)
	}

	// Build a lookup map by employee number.
	type empData struct {
		LastName  string
		FirstName string
		MagCardID int
	}
	lookup := make(map[int]empData)
	for _, rec := range f.Records() {
		id := dbf.GetInt(rec, "EMP_NUMBER")
		if id == 0 {
			continue
		}
		status := strings.TrimSpace(dbf.GetString(rec, "EMP_STATUS"))
		// Skip inactive employees if status is present
		if strings.EqualFold(status, "I") {
			continue
		}
		lookup[id] = empData{
			LastName:  strings.TrimSpace(dbf.GetString(rec, "LAST_NAME")),
			FirstName: strings.TrimSpace(dbf.GetString(rec, "FIRST_NAME")),
			MagCardID: dbf.GetInt(rec, "CARD_NUM"),
		}
	}

	for i, e := range employees {
		if d, ok := lookup[e.ID]; ok {
			if d.LastName != "" {
				employees[i].LastName = d.LastName
			}
			if d.FirstName != "" {
				employees[i].FirstName = d.FirstName
			}
			if d.MagCardID != 0 {
				employees[i].MagCardID = d.MagCardID
			}
		}
	}
	log.Printf("employees: enriched from EMPFILE.DBF (%d entries)", len(lookup))
	return employees, nil
}
