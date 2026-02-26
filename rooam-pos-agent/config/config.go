// Package config reads the rooam_config.json file produced by the Inno Setup
// installer and derives the SC, DBF, and ALTDBF directory paths.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Location holds venue address information from the config file.
type Location struct {
	Name     string `json:"name"`
	Country  string `json:"country"`
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// Rooam holds Rooam-specific identifiers from the config file.
type Rooam struct {
	TenderID   string `json:"tender_id"`
	EmployeeID string `json:"employee_id"`
}

// POSitouch holds the raw POSitouch paths from the config file.
type POSitouch struct {
	SpcwinPath     string `json:"spcwin_path"`
	VirtualSection string `json:"virtual_section"`
	XMLSection     string `json:"xml_section"`
}

// Config is the top-level configuration structure.
type Config struct {
	Location  Location  `json:"location"`
	Rooam     Rooam     `json:"rooam"`
	POSitouch POSitouch `json:"positouch"`

	// Derived paths (not from JSON)
	SCPath     string
	DBFPath    string
	ALTDBFPath string
}

// Load reads rooam_config.json from the given path and returns a populated
// Config with derived directory paths.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}

	if cfg.POSitouch.SpcwinPath == "" {
		return nil, fmt.Errorf("config: positouch.spcwin_path is empty")
	}

	// SC path is the directory containing SPCWIN.ini
	scDir := filepath.Dir(cfg.POSitouch.SpcwinPath)

	// DBF and ALTDBF are siblings of the SC directory
	parentDir := filepath.Dir(scDir)

	cfg.SCPath = scDir + string(filepath.Separator)
	cfg.DBFPath = filepath.Join(parentDir, "DBF") + string(filepath.Separator)
	cfg.ALTDBFPath = filepath.Join(parentDir, "ALTDBF") + string(filepath.Separator)

	return &cfg, nil
}
