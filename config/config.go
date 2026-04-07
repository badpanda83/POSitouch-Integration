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

// StripeConfig holds Stripe billing settings used by the subscription server.
// All fields can be supplied via JSON config or overridden by environment variables.
type StripeConfig struct {
	// SecretKey is the Stripe secret API key (sk_live_... or sk_test_...).
	// Override with STRIPE_SECRET_KEY env var.
	SecretKey string `json:"secret_key,omitempty"`

	// WebhookSecret is the Stripe webhook signing secret (whsec_...).
	// Override with STRIPE_WEBHOOK_SECRET env var.
	WebhookSecret string `json:"webhook_secret,omitempty"`

	// PriceID is the Stripe Price ID for the $5/mo subscription (price_...).
	// Override with STRIPE_PRICE_ID env var.
	PriceID string `json:"price_id,omitempty"`

	// SuccessURL is where Stripe redirects after a successful checkout.
	// Override with STRIPE_SUCCESS_URL env var.
	SuccessURL string `json:"success_url,omitempty"`

	// CancelURL is where Stripe redirects if the customer cancels checkout.
	// Override with STRIPE_CANCEL_URL env var.
	CancelURL string `json:"cancel_url,omitempty"`

	// Port is the port the Stripe webhook/checkout server listens on (default 3000).
	Port string `json:"port,omitempty"`
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

	// Stripe holds billing/subscription settings for the Stripe server component.
	// All fields are optional and can be provided via environment variables.
	Stripe *StripeConfig `json:"stripe,omitempty"`

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

// MICROS3700Config holds connection settings for the MICROS 3700 Sybase ODBC interface.
type MICROS3700Config struct {
	ODBCDSN          string `json:"odbc_dsn,omitempty"`           // default "Micros"
	RevenueCenterID  int    `json:"revenue_center_id,omitempty"`
	TerminalID       int    `json:"terminal_id,omitempty"`
	// Deprecated: these fields are retained for backward compatibility but are no longer used.
	TransactionServicesURL string `json:"transaction_services_url,omitempty"`
	HTTPUser               string `json:"http_user,omitempty"`
	HTTPPassword           string `json:"http_password,omitempty"`
	ConnectionString       string `json:"connection_string,omitempty"`
	DatabaseHost           string `json:"database_host,omitempty"`
	DatabaseName           string `json:"database_name,omitempty"`
	DatabaseUser           string `json:"database_user,omitempty"`
	DatabasePassword       string `json:"database_password,omitempty"`
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

	// Populate Stripe config from environment variables when not set in JSON.
	// This allows secrets to be injected at runtime (e.g. Railway env vars)
	// without committing them to rooam_config.json.
	if cfg.Stripe == nil {
		cfg.Stripe = &StripeConfig{}
	}
	if cfg.Stripe.SecretKey == "" {
		cfg.Stripe.SecretKey = os.Getenv("STRIPE_SECRET_KEY")
	}
	if cfg.Stripe.WebhookSecret == "" {
		cfg.Stripe.WebhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
	}
	if cfg.Stripe.PriceID == "" {
		cfg.Stripe.PriceID = os.Getenv("STRIPE_PRICE_ID")
	}
	if cfg.Stripe.SuccessURL == "" {
		cfg.Stripe.SuccessURL = os.Getenv("STRIPE_SUCCESS_URL")
	}
	if cfg.Stripe.CancelURL == "" {
		cfg.Stripe.CancelURL = os.Getenv("STRIPE_CANCEL_URL")
	}
	if cfg.Stripe.Port == "" {
		if p := os.Getenv("STRIPE_SERVER_PORT"); p != "" {
			cfg.Stripe.Port = p
		} else {
			cfg.Stripe.Port = "3000"
		}
	}
	// If no Stripe credentials are set at all, nil out the struct so callers
	// can use cfg.Stripe == nil as "Stripe not configured".
	if cfg.Stripe.SecretKey == "" && cfg.Stripe.WebhookSecret == "" {
		cfg.Stripe = nil
	}

	return &cfg, nil
}
