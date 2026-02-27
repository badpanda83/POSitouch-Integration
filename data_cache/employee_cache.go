package cache

import (
	"encoding/json"
	"os"
)

func WriteEmployeesToCache(employees []Employee, path string) error {
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	return json.NewEncoder(f).Encode(employees)
}

func ReadEmployeesFromCache(path string) ([]Employee, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var employees []Employee
	if err := json.NewDecoder(f).Decode(&employees); err != nil { return nil, err }
	return employees, nil
}