package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Server is the Stripe billing HTTP server.
// It exposes:
//
//	POST /webhook  — receives and verifies Stripe webhook events
//	POST /checkout — creates a Stripe Checkout Session and returns its URL
//	GET  /health   — liveness probe
type Server struct {
	// StripeSecretKey is the Stripe secret API key (sk_live_... or sk_test_...).
	StripeSecretKey string

	// WebhookSecret is the Stripe webhook endpoint signing secret (whsec_...).
	WebhookSecret string

	// PriceID is the Stripe Price ID for the subscription product (price_...).
	PriceID string

	// SuccessURL is the URL Stripe redirects to after a successful payment.
	SuccessURL string

	// CancelURL is the URL Stripe redirects to if the customer cancels.
	CancelURL string

	// CloudEndpoint is the Rooam cloud server base URL used to activate/deactivate tenants.
	CloudEndpoint string

	// CloudAPIKey is the Rooam cloud server API key used to authenticate tenant updates.
	CloudAPIKey string

	// Port is the TCP port the server listens on (default "3000").
	Port string
}

// stripeCheckoutResponse is the subset of the Stripe CheckoutSession response
// we care about.
type stripeCheckoutResponse struct {
	URL string `json:"url"`
}

// stripeEvent represents the top-level Stripe webhook event payload.
type stripeEvent struct {
	Type string          `json:"type"`
	Data stripeEventData `json:"data"`
}

type stripeEventData struct {
	Object stripeEventObject `json:"object"`
}

type stripeEventObject struct {
	// Used in checkout.session.completed
	Metadata map[string]string `json:"metadata"`
	// Used in customer.subscription.deleted / invoice.payment_failed
	CustomerEmail string `json:"customer_email"`
}

// Start registers the HTTP handlers and begins listening on s.Port.
// It blocks until the server exits (call in a goroutine from main).
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/checkout", s.handleCheckout)
	mux.HandleFunc("/health", s.handleHealth)

	port := s.Port
	if port == "" {
		port = "3000"
	}
	addr := ":" + port
	log.Printf("[billing] Stripe billing server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

// handleHealth responds to liveness probes.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status":"ok"}`)
}

// handleCheckout creates a Stripe Checkout Session for the given location_id
// and returns the hosted checkout URL. The client (installer wizard or web
// portal) can then open a browser to that URL.
//
// Request body (JSON):
//
//	{ "location_id": "my-restaurant", "email": "owner@example.com" }
//
// Response body (JSON):
//
//	{ "url": "https://checkout.stripe.com/c/pay/..." }
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		LocationID string `json:"location_id"`
		Email      string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.LocationID == "" {
		http.Error(w, "location_id is required", http.StatusBadRequest)
		return
	}

	checkoutURL, err := s.createCheckoutSession(req.LocationID, req.Email)
	if err != nil {
		log.Printf("[billing] createCheckoutSession error: %v", err)
		http.Error(w, "failed to create checkout session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": checkoutURL})
}

// handleWebhook verifies the Stripe-Signature header and processes the event.
// Only checkout.session.completed, customer.subscription.deleted, and
// invoice.payment_failed are acted upon; all others are acknowledged silently.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the raw body — Stripe requires the exact bytes for signature verification.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB limit
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Verify webhook signature before trusting any payload.
	if err := s.verifyStripeSignature(r.Header.Get("Stripe-Signature"), body); err != nil {
		log.Printf("[billing] webhook signature verification failed: %v", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var event stripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid event JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[billing] received Stripe event: %s", event.Type)

	switch event.Type {
	case "checkout.session.completed":
		locationID := event.Data.Object.Metadata["location_id"]
		if locationID == "" {
			log.Printf("[billing] checkout.session.completed: missing location_id in metadata")
			break
		}
		if err := ActivateTenant(s.CloudEndpoint, s.CloudAPIKey, locationID); err != nil {
			log.Printf("[billing] ActivateTenant(%q) error: %v", locationID, err)
		} else {
			log.Printf("[billing] tenant %q activated", locationID)
		}

	case "customer.subscription.deleted", "invoice.payment_failed":
		locationID := event.Data.Object.Metadata["location_id"]
		if locationID == "" {
			log.Printf("[billing] %s: missing location_id in metadata", event.Type)
			break
		}
		if err := DeactivateTenant(s.CloudEndpoint, s.CloudAPIKey, locationID); err != nil {
			log.Printf("[billing] DeactivateTenant(%q) error: %v", locationID, err)
		} else {
			log.Printf("[billing] tenant %q deactivated", locationID)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// verifyStripeSignature verifies the Stripe-Signature header against the raw
// webhook payload using HMAC-SHA256 as described in the Stripe documentation:
// https://stripe.com/docs/webhooks/signatures
func (s *Server) verifyStripeSignature(sigHeader string, body []byte) error {
	if s.WebhookSecret == "" {
		return fmt.Errorf("webhook secret is not configured")
	}
	if sigHeader == "" {
		return fmt.Errorf("Stripe-Signature header is missing")
	}

	// The header format is: "t=<timestamp>,v1=<sig1>,v1=<sig2>,..."
	var timestamp string
	var signatures []string

	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}

	if timestamp == "" {
		return fmt.Errorf("timestamp missing from Stripe-Signature header")
	}
	if len(signatures) == 0 {
		return fmt.Errorf("no v1 signatures found in Stripe-Signature header")
	}

	// Reject events with timestamps more than 5 minutes old to prevent replay attacks.
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp in Stripe-Signature: %w", err)
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return fmt.Errorf("webhook timestamp too old — possible replay attack")
	}

	// Compute expected signature: HMAC-SHA256(secret, "<timestamp>.<body>")
	mac := hmac.New(sha256.New, []byte(s.WebhookSecret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return nil
		}
	}
	return fmt.Errorf("webhook signature mismatch")
}

// createCheckoutSession calls the Stripe API to create a hosted Checkout Session
// and returns the session URL. The location_id is stored in the session metadata
// so that the checkout.session.completed webhook can identify the tenant.
//
// Multi-currency is handled automatically by Stripe when the customer's currency
// is detected from their browser locale; no explicit currency configuration is
// needed beyond setting up prices in the Stripe Dashboard.
func (s *Server) createCheckoutSession(locationID, email string) (string, error) {
	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("line_items[0][price]", s.PriceID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("metadata[location_id]", locationID)
	if email != "" {
		form.Set("customer_email", email)
	}
	successURL := s.SuccessURL
	if successURL == "" {
		successURL = "https://example.com/success"
	}
	cancelURL := s.CancelURL
	if cancelURL == "" {
		cancelURL = "https://example.com/cancel"
	}
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)

	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.stripe.com/v1/checkout/sessions",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("createCheckoutSession: build request: %w", err)
	}
	req.SetBasicAuth(s.StripeSecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("createCheckoutSession: stripe request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("createCheckoutSession: Stripe returned %s: %s", resp.Status, string(body))
	}

	var session stripeCheckoutResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", fmt.Errorf("createCheckoutSession: decode response: %w", err)
	}
	if session.URL == "" {
		return "", fmt.Errorf("createCheckoutSession: Stripe response missing url field")
	}
	return session.URL, nil
}
