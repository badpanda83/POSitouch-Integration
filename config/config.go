package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	InstallDir = `C:\Program Files\Rooam\POSitouch`
	ConfigFile = `rooam_config.json`
	CacheFile  = `rooam_cache.json`
	LogFile    = `rooam_agent.log`
)

// Location holds the physical location details of the venue.
type Location struct {
	Name     string `json:"name"`
	Country  string `json:"country"`
	Address1 string `json:"address1"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// Rooam holds Rooam-specific IDs used for integration.
type Rooam struct {
	TenderID   string `json:"tender_id"`
	EmployeeID string `json:"employee_id"`
}

// POSitouch holds POSitouch-specific configuration.
type POSitouch struct {
	SpcwinPath     string `json:"spcwin_path"`
	VirtualSection string `json:"virtual_section"`
	XMLSection     string `json:"xml_section"`
}

// Config is the top-level configuration loaded from rooam_config.json.
type Config struct {
	Location  Location  `json:"location"`
	Rooam     Rooam     `json:"rooam"`
	POSitouch POSitouch `json:"positouch"`

	// Derived fields — populated by Load, not from JSON.
	DBFDir    string `json:"-"`
	AltDBFDir string `json:"-"`
	LogPath   string `json:"-"`
	CachePath string `json:"-"`
}

// Load reads rooam_config.json from the install directory and derives
// the DBF directory path from the spcwin_path field.
func Load() (*Config, error) {
	configPath := filepath.Join(InstallDir, ConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.deriveDBFDir()
	cfg.LogPath = filepath.Join(InstallDir, LogFile)
	cfg.CachePath = filepath.Join(InstallDir, CacheFile)
	return &cfg, nil
}

// deriveDBFDir uses the spcwin_path from config to locate the DBF directory.
// If spcwin_path is "C:\SC\SPCWIN.ini", DBF is at "C:\DBF\" and ALTDBF at "C:\ALTDBF\".
// Falls back to scanning drives A-Z for \DBF\NAMES.DBF if derivation fails.
func (c *Config) deriveDBFDir() {
	if c.POSitouch.SpcwinPath != "" {
		drive := driveLetter(c.POSitouch.SpcwinPath)
		if drive != "" {
			dbfDir := drive + `:\DBF`
			if dirExists(dbfDir) {
				c.DBFDir = dbfDir
				c.AltDBFDir = drive + `:\ALTDBF`
				return
			}
		}
	}

	// Fallback: scan drives A-Z.
	for _, d := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		candidate := string(d) + `:\DBF`
		if fileExists(filepath.Join(candidate, "NAMES.DBF")) {
			c.DBFDir = candidate
			c.AltDBFDir = string(d) + `:\ALTDBF`
			return
		}
	}

	// Last resort: use the path derived from spcwin_path even if the directory
	// doesn't exist yet (POSIDBFW may not have run).
	if c.POSitouch.SpcwinPath != "" {
		drive := driveLetter(c.POSitouch.SpcwinPath)
		if drive != "" {
			c.DBFDir = drive + `:\DBF`
			c.AltDBFDir = drive + `:\ALTDBF`
		}
	}
}

// driveLetter extracts the single drive letter from an absolute Windows path
// such as "C:\SC\SPCWIN.ini" → "C".  Returns "" if the path has no drive.
func driveLetter(path string) string {
	if len(path) >= 2 && path[1] == ':' {
		return strings.ToUpper(string(path[0]))
	}
	return ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
