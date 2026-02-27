package cache

import (
	"encoding/json"
	"os"
)

func WriteOrderTypesToCache(orderTypes []OrderType, path string) error {
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	return json.NewEncoder(f).Encode(orderTypes)
}

func ReadOrderTypesFromCache(path string) ([]OrderType, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var orderTypes []OrderType
	if err := json.NewDecoder(f).Decode(&orderTypes); err != nil { return nil, err }
	return orderTypes, nil
}