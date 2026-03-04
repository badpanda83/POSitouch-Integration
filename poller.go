package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/ordering"
)

func pollPendingOrders(cfg *config.Config, xmlInOrderDir string) {
	base := strings.TrimRight(cfg.Cloud.Endpoint, "/")
	locationID := cfg.LocationID
	if locationID == "" {
		locationID = cfg.Location.Name
	}

	// Build request using url.URL directly to avoid Go encoding the apostrophe.
	// url.Parse on a string with ' will keep it literal; we then pass it to
	// http.NewRequest via its String() which also keeps it literal.
	rawURL := fmt.Sprintf("%s/%s/tickets/pending", base, locationID)
	log.Printf("[poller] polling URL: %s", rawURL)

	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("[poller] error parsing URL: %%v", err)
		return
	}
	log.Printf("[poller] parsed URL: %s", parsed.String())

	req, err := http.NewRequest("GET", parsed.String(), nil)
	if err != nil {
		log.Printf("[poller] error building request: %%v", err)
		return
	}
	if cfg.Cloud.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Cloud.APIKey)
	}
	log.Printf("[poller] request URL after NewRequest: %s", req.URL.String())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[poller] error contacting cloud server: %%v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("[poller] response status: %s", resp.Status)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[poller] unexpected status from cloud server: %s", resp.Status)
		return
	}

	var pending []struct {
		ReferenceNumber string          `json:"reference_number"`
		LocationID      string          `json:"location_id"`
		Payload         json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pending); err != nil {
		log.Printf("[poller] error decoding response: %%v", err)
		return
	}
	log.Printf("[poller] decoded %d pending order(s)", len(pending))
	if len(pending) == 0 {
		return
	}
	log.Printf("[poller] received %d pending order(s)", len(pending))

	for _, p := range pending {
		var ticketReq ordering.CreateTicketRequest
		if err := json.Unmarshal(p.Payload, &ticketReq); err != nil {
			log.Printf("[poller] error unmarshalling order %s: %%v", p.ReferenceNumber, err)
			continue
		}
		if err := ordering.WriteOrderXML(ticketReq, xmlInOrderDir); err != nil {
			log.Printf("[poller] error writing XML for order %s: %%v", p.ReferenceNumber, err)
		} else {
			log.Printf("[poller] wrote XML for order ref=%s", p.ReferenceNumber)
		}
	}
}