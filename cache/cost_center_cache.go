package cache

import (
	"encoding/json"
	"os"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

func WriteCostCentersToCache(costCenters []positouch.CostCenter, path string) error {
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	return json.NewEncoder(f).Encode(costCenters)
}

func ReadCostCentersFromCache(path string) ([]positouch.CostCenter, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var costCenters []positouch.CostCenter
	if err := json.NewDecoder(f).Decode(&costCenters); err != nil { return nil, err }
	return costCenters, nil
}