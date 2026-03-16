package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultConfigPath = "rooam_config.json"

// ----- Location definition -----
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

// ----- Rooam-specific fields -----
type Rooam struct {
	TenderID   string `json:"tender_id"`
	EmployeeID string `json:"employee_id"`
}

// ----- POSitouch fields -----
type POSitouch struct {
	SpcwinPath     string `json:"spcwin_path"`
	VirtualSection string `json:"virtual_section"`
	XMLSection     string `json:"xml_section"`
}

// ----- Cloud config -----
type CloudConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key"`
}

// OAuthConfig holds settings for OAuth 2.0 / OIDC authentication.
// Only used when auth_mode == "oauth". Currently stubbed — see TODO(phase-3b).
type OAuthConfig struct {
	ProviderURL       string   `json:"provider_url"`        // e.g. https://your-company.okta.com/oauth2/default
	ClientID          string   `json:"client_id"`
	ClientSecret      string   `json:"client_secret"`
	Scopes            []string `json:"scopes"`
	TokenRefreshHours int      `json:"token_refresh_hours"` // force re-auth interval (0 = IdP expiry only)
}

// ----- Top-level Config -----
type Config struct {
	Location  Location    `json:"location"`
	Rooam     Rooam       `json:"rooam"`
	POSitouch POSitouch   `json:"positouch"`
	Cloud     CloudConfig `json:"cloud"`

	// AuthMode selects the authentication provider. Values: "static" (default) | "oauth"
	// TODO(phase-3b): set to "oauth" once OAuthProvider is implemented.
	AuthMode string       `json:"auth_mode,omitempty"`
	OAuth    *OAuthConfig `json:"oauth,omitempty"`

	// POSType selects which driver to load. Values: "positouch" | "micros3700"
	POSType string `json:"pos_type"`

	// MICROS3700 holds connection settings for the MICROS 3700 Transaction Services
	// interface. Only used when POSType == "micros3700".
	MICROS3700 *MICROS3700Config `json:"micros3700,omitempty"`

	XMLDir        string `json:"xml_dir"`         // open tickets directory
	XMLCloseDir   string `json:"xml_close_dir"`   // closed tickets directory
	XMLInOrderDir string `json:"xml_inorder_dir"` // inbound order drop directory

	CloudServerURL string `json:"cloud_server_url"` // base URL of the Railway cloud server
	LocationID     string `json:"location_id"`      // location identifier used with the cloud server

	SCDir      string
	SCPath     string
	DBFDir     string
	DBFPath    string
	ALTDBFDir  string
	ALTDBFPath string
	AltDBFDir  string // (CamelCase for main.go compatibility)
	InstallDir string // Directory containing config file
}

// MICROS3700Config holds connection settings for the MICROS 3700 Transaction Services interface.
type MICROS3700Config struct {
	TransactionServicesURL string `json:"transaction_services_url"`
	HTTPUser               string `json:"http_user,omitempty"`
	HTTPPassword           string `json:"http_password,omitempty"`
	ConnectionString       string `json:"connection_string,omitempty"`
	DatabaseHost           string `json:"database_host"`
	DatabaseName           string `json:"database_name"`
	DatabaseUser           string `json:"database_user"`
	DatabasePassword       string `json:"database_password"`
	RevenueCenterID        int    `json:"revenue_center_id,omitempty"`
	TerminalID             int    `json:"terminal_id,omitempty"`
}

// EffectivePOSType returns the pos_type, defaulting to "positouch" for backwards
// compatibility with existing configs that don't have the field set.
func (c *Config) EffectivePOSType() string {
	if c.POSType == "" {
		return "positouch"
	}
	return c.POSType
}

// Load reads the config JSON file and computes the paths used by the agent.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}

	if cfg.EffectivePOSType() == "positouch" {
		if cfg.POSitouch.SpcwinPath == "" {
			return nil, fmt.Errorf("config: positouch.spcwin_path is empty")
		}

		scDir := filepath.Dir(cfg.POSitouch.SpcwinPath)
		parentDir := filepath.Dir(scDir)
		dbfDir := filepath.Join(parentDir, "DBF")
		altdbfDir := filepath.Join(parentDir, "ALTDBF")

		cfg.SCPath = scDir + string(filepath.Separator)
		cfg.SCDir = cfg.SCPath
		cfg.DBFPath = dbfDir + string(filepath.Separator)
		cfg.DBFDir = cfg.DBFPath
		cfg.ALTDBFPath = altdbfDir + string(filepath.Separator)
		cfg.ALTDBFDir = cfg.ALTDBFPath
		cfg.AltDBFDir = cfg.ALTDBFDir
	}

	if cfg.EffectivePOSType() == "micros3700" {
		if cfg.MICROS3700 == nil {
			return nil, fmt.Errorf("config: micros3700 configuration block is required")
		}
		if cfg.MICROS3700.TransactionServicesURL == "" {
			return nil, fmt.Errorf("config: micros3700.transaction_services_url is required")
		}
		if cfg.MICROS3700.DatabaseName == "" {
			return nil, fmt.Errorf("config: micros3700.database_name is required")
		}
		if cfg.MICROS3700.DatabaseUser == "" {
			return nil, fmt.Errorf("config: micros3700.database_user is required")
		}
	}

	cfg.InstallDir = filepath.Dir(path)

	if cfg.CloudServerURL == "" {
		cfg.CloudServerURL = os.Getenv("CLOUD_SERVER_URL")
	}
	if cfg.LocationID == "" {
		cfg.LocationID = os.Getenv("LOCATION_ID")
	}
	if cfg.LocationID == "" {
		cfg.LocationID = cfg.Location.Name
	}

	return &cfg, nil
}
