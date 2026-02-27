package cache

import (
	"encoding/json"
	"os"
)

func WriteTendersToCache(tenders []Tender, path string) error {
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	return json.NewEncoder(f).Encode(tenders)
}

func ReadTendersFromCache(path string) ([]Tender, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var tenders []Tender
	if err := json.NewDecoder(f).Decode(&tenders); err != nil { return nil, err }
	return tenders, nil
}