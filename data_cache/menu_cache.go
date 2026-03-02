package cache

import (
	"encoding/json"
	"os"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

func WriteMenuItemsToCache(menuItems []positouch.MenuItem, path string) error {
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	return json.NewEncoder(f).Encode(menuItems)
}

func ReadMenuItemsFromCache(path string) ([]positouch.MenuItem, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var items []positouch.MenuItem
	if err := json.NewDecoder(f).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}