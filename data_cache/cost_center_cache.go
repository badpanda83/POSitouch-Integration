package cache

import (
	"encoding/json"
	"os"
)

func WriteCostCentersToCache(costCenters []CostCenter, path string) error {
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	return json.NewEncoder(f).Encode(costCenters)
}