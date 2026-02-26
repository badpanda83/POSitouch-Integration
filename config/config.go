// Package config reads the rooam_config.json installer configuration file
// and derives the SC, DBF, and ALTDBF directory paths.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Location holds venue address and contact information.
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

// Rooam holds Rooam-specific configuration identifiers.
type Rooam struct {
	TenderID   string `json:"tender_id"`
	EmployeeID string `json:"employee_id"`
}

// POSitouch holds POSitouch-specific paths and section names.
type POSitouch struct {
	SpcwinPath     string `json:"spcwin_path"`
	VirtualSection string `json:"virtual_section"`
	XMLSection     string `json:"xml_section"`
}

// Config is the top-level structure for rooam_config.json.
type Config struct {
	Location  Location  `json:"location"`
	Rooam     Rooam     `json:"rooam"`
	POSitouch POSitouch `json:"positouch"`

	// Derived paths — not from JSON.
	SCPath     string `json:"-"`
	DBFPath    string `json:"-"`
	ALTDBFPath string `json:"-"`
}

// LoadConfig reads the JSON config file at path, derives SC/DBF/ALTDBF paths,
// and returns the populated Config.
func LoadConfig(path string) (*Config, error) {
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

	// SC path is the directory that contains SPCWIN.ini
	cfg.SCPath = filepath.Dir(cfg.POSitouch.SpcwinPath)

	// DBF and ALTDBF are siblings of the SC directory
	parent := filepath.Dir(cfg.SCPath)
	cfg.DBFPath = filepath.Join(parent, "DBF")
	cfg.ALTDBFPath = filepath.Join(parent, "ALTDBF")

	return &cfg, nil
}
