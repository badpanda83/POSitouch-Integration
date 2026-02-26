// Package config reads the rooam_config.json file and derives POSitouch paths.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RooamConfig is the top-level configuration structure loaded from rooam_config.json.
type RooamConfig struct {
	Location  LocationConfig  `json:"location"`
	Rooam     RooamSettings   `json:"rooam"`
	POSitouch POSitouchConfig `json:"positouch"`
}

// LocationConfig holds venue/location information.
type LocationConfig struct {
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

// RooamSettings holds Rooam-specific configuration.
type RooamSettings struct {
	TenderID   string `json:"tender_id"`
	EmployeeID string `json:"employee_id"`
}

// POSitouchConfig holds POSitouch-specific paths and settings.
type POSitouchConfig struct {
	SPCWINPath     string `json:"spcwin_path"`
	VirtualSection string `json:"virtual_section"`
	XMLSection     string `json:"xml_section"`
}

// LoadConfig reads and parses the rooam_config.json file at path.
func LoadConfig(path string) (*RooamConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	var cfg RooamConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return &cfg, nil
}

// DriveLetter returns the drive letter (e.g. "C:") derived from spcwin_path.
// If spcwin_path is empty or has no drive letter, returns "C:".
func (c *RooamConfig) DriveLetter() string {
	p := c.POSitouch.SPCWINPath
	if len(p) >= 2 && p[1] == ':' {
		return strings.ToUpper(string(p[0])) + ":"
	}
	return "C:"
}

// SCDir returns the SC directory path (e.g. "C:\SC\").
func (c *RooamConfig) SCDir() string {
	return c.DriveLetter() + `\SC\`
}

// DBFDir returns the DBF directory path (e.g. "C:\DBF\").
func (c *RooamConfig) DBFDir() string {
	return c.DriveLetter() + `\DBF\`
}

// ALTDBFDir returns the ALTDBF directory path (e.g. "C:\ALTDBF\").
func (c *RooamConfig) ALTDBFDir() string {
	return c.DriveLetter() + `\ALTDBF\`
}

// findFile returns the first path (case-insensitively) that matches filename
// inside dir. Returns an empty string if not found.
func FindFile(dir, filename string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	upper := strings.ToUpper(filename)
	for _, e := range entries {
		if strings.ToUpper(e.Name()) == upper {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}
