package cache

import (
	"encoding/json"
	"os"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

func WriteEmployeesToCache(employees []positouch.Employee, path string) error {
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	return json.NewEncoder(f).Encode(employees)
}

func ReadEmployeesFromCache(path string) ([]positouch.Employee, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var employees []positouch.Employee
	if err := json.NewDecoder(f).Decode(&employees); err != nil { return nil, err }
	return employees, nil
}