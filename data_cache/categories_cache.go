package cache

import (
	"encoding/json"
	"os"

	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// WriteCategoriesToCache serializes the categories slice to the given JSON file.
func WriteCategoriesToCache(categories []positouch.MenuCategory, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(categories)
}

// ReadCategoriesFromCache loads categories from the given JSON file.
func ReadCategoriesFromCache(path string) ([]positouch.MenuCategory, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var categories []positouch.MenuCategory
	if err := json.NewDecoder(f).Decode(&categories); err != nil {
		return nil, err
	}
	return categories, nil
}