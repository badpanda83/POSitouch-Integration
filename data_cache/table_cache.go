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

func ReadTablesFromCache(path string) ([]Table, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var tables []Table
	if err := json.NewDecoder(f).Decode(&tables); err != nil { return nil, err }
	return tables, nil
}