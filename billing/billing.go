// Package billing provides helpers for activating and deactivating tenants on
// the Rooam cloud server when Stripe subscription events are received.
package billing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// httpClient is the shared HTTP client used for cloud server API calls.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// ActivateTenant marks a location as active on the cloud server.
// It should be called when a Stripe checkout.session.completed event is received.
func ActivateTenant(cloudEndpoint, apiKey, locationID string) error {
	return setActive(cloudEndpoint, apiKey, locationID, true)
}

// DeactivateTenant marks a location as inactive on the cloud server.
// It should be called when a Stripe customer.subscription.deleted or
// invoice.payment_failed event is received.
func DeactivateTenant(cloudEndpoint, apiKey, locationID string) error {
	return setActive(cloudEndpoint, apiKey, locationID, false)
}

// setActive sends a PATCH request to the cloud server to update the active flag
// for the given location.
func setActive(cloudEndpoint, apiKey, locationID string, active bool) error {
	base := strings.TrimRight(cloudEndpoint, "/")
	url := fmt.Sprintf("%s/tenants/%s", base, locationID)

	payload, err := json.Marshal(map[string]bool{"active": active})
	if err != nil {
		return fmt.Errorf("billing: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("billing: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("billing: request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("billing: cloud server returned %s for PATCH %s", resp.Status, url)
	}
	return nil
}

// RegisterTenant creates an inactive tenant entry on the cloud server and
// returns a Stripe checkout URL. locationID is used as the tenant identifier
// and is stored in the Stripe session metadata so that webhook events can
// reference the correct tenant.
//
// enrollURL is the public URL of the billing server's POST /checkout endpoint.
func RegisterTenant(enrollURL, locationID, email string) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"location_id": locationID,
		"email":       email,
	})
	if err != nil {
		return "", fmt.Errorf("billing: marshal register payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, enrollURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("billing: create register request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("billing: register request to %s failed: %w", enrollURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("billing: enrollment server returned %s", resp.Status)
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("billing: decode register response: %w", err)
	}
	if result.URL == "" {
		return "", fmt.Errorf("billing: enrollment response missing checkout URL")
	}
	return result.URL, nil
}
