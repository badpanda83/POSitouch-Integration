package micros3700driver

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/badpanda83/POSitouch-Integration/config"
)

// postXML sends an HTTP POST request with an XML body to the MICROS Transaction
// Services endpoint, using HTTP Basic Auth. It returns the raw response body.
func postXML(cfg *config.MICROS3700Config, payload interface{}) ([]byte, error) {
	body, err := xml.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("micros3700: marshal XML: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, cfg.TransactionServicesURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("micros3700: create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	if cfg.HTTPUser != "" {
		req.SetBasicAuth(cfg.HTTPUser, cfg.HTTPPassword)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("micros3700: HTTP POST to %s: %w", cfg.TransactionServicesURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("micros3700: read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("micros3700: unexpected HTTP status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
