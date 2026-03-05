package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/badpanda83/POSitouch-Integration/config"
)

// runConnectivityCheck tests cloud and POS reachability.
// It prints a pass/fail line for each test and returns nil if all critical
// tests pass (non-critical issues are printed as warnings).
func runConnectivityCheck(cfg *config.Config) error {
	client := &http.Client{Timeout: 5 * time.Second}
	var errs []string

	// 1. Cloud server reachability — GET <endpoint>/health (or just <endpoint>).
	healthURL := strings.TrimRight(cfg.Cloud.Endpoint, "/") + "/health"
	resp, err := client.Get(healthURL)
	if err != nil {
		// Fall back to the base endpoint.
		resp, err = client.Get(cfg.Cloud.Endpoint)
	}
	if err != nil {
		fmt.Printf("[connectivity] ✗ cloud reachability (%s): %v\n", cfg.Cloud.Endpoint, err)
		errs = append(errs, fmt.Sprintf("cloud unreachable: %v", err))
	} else {
		resp.Body.Close()
		fmt.Printf("[connectivity] ✓ cloud reachability: HTTP %d\n", resp.StatusCode)
	}

	// 2. Cloud auth check — GET <endpoint>/<locationID>/tickets with Bearer token.
	locationID := cfg.LocationID
	if locationID == "" {
		locationID = cfg.Location.Name
	}
	authURL := strings.TrimRight(cfg.Cloud.Endpoint, "/") + "/" + locationID + "/tickets"
	req, err := http.NewRequest(http.MethodGet, authURL, nil)
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+cfg.Cloud.APIKey)
		authResp, authErr := client.Do(req)
		if authErr != nil {
			fmt.Printf("[connectivity] ✗ cloud auth check: %v\n", authErr)
			errs = append(errs, fmt.Sprintf("cloud auth check failed: %v", authErr))
		} else {
			authResp.Body.Close()
			switch authResp.StatusCode {
			case http.StatusUnauthorized:
				fmt.Printf("[connectivity] ⚠ cloud auth check: HTTP 401 — API key may be invalid\n")
			case http.StatusOK, http.StatusNotFound:
				fmt.Printf("[connectivity] ✓ cloud auth check: HTTP %d\n", authResp.StatusCode)
			default:
				fmt.Printf("[connectivity] ✓ cloud auth check: HTTP %d\n", authResp.StatusCode)
			}
		}
	}

	posType := cfg.EffectivePOSType()

	// 3. MICROS 3700 Transaction Services reachability.
	if posType == "micros3700" && cfg.MICROS3700 != nil {
		tsURL := cfg.MICROS3700.TransactionServicesURL
		tsResp, tsErr := client.Get(tsURL)
		if tsErr != nil {
			fmt.Printf("[connectivity] ✗ MICROS 3700 Transaction Services (%s): %v\n", tsURL, tsErr)
			errs = append(errs, fmt.Sprintf("MICROS 3700 unreachable: %v", tsErr))
		} else {
			tsResp.Body.Close()
			fmt.Printf("[connectivity] ✓ MICROS 3700 Transaction Services: HTTP %d\n", tsResp.StatusCode)
		}
	}

	// 4. POSitouch XML directory write test.
	if posType == "positouch" && cfg.XMLInOrderDir != "" {
		tmp, err := os.CreateTemp(cfg.XMLInOrderDir, ".rooam-connectivity-*")
		if err != nil {
			fmt.Printf("[connectivity] ✗ xml_inorder_dir write test (%s): %v\n", cfg.XMLInOrderDir, err)
			errs = append(errs, fmt.Sprintf("xml_inorder_dir not writable: %v", err))
		} else {
			tmp.Close()
			_ = os.Remove(tmp.Name())
			fmt.Printf("[connectivity] ✓ xml_inorder_dir write test: %s\n", cfg.XMLInOrderDir)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("connectivity checks failed: %s", joinErrors(errs))
	}
	return nil
}
