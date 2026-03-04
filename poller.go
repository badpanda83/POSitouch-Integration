package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/ordering"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const (
	pollerConfirmTimeout     = 30 * time.Second
	pollerConfirmInterval    = 2 * time.Second
	pollerTicketMatchWindow  = 60 * time.Second
)

func pollPendingOrders(cfg *config.Config, xmlInOrderDir string, xmlDir string, xmlCloseDir string) {
	base := strings.TrimRight(cfg.Cloud.Endpoint, "/")
	locationID := cfg.LocationID
	if locationID == "" {
		locationID = cfg.Location.Name
	}

	rawURL := fmt.Sprintf("%s/%s/tickets/pending", base, locationID)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("[poller] error parsing URL: %v", err)
		return
	}

	req, err := http.NewRequest("GET", parsed.String(), nil)
	if err != nil {
		log.Printf("[poller] error building request: %v", err)
		return
	}
	if cfg.Cloud.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Cloud.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[poller] error contacting cloud server: %v", err)
		return
	}
	defer resp.Body.Close()

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
		log.Printf("[poller] error decoding response: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	log.Printf("[poller] received %d pending order(s)", len(pending))

	for _, p := range pending {
		var ticketReq ordering.CreateTicketRequest
		if err := json.Unmarshal(p.Payload, &ticketReq); err != nil {
			log.Printf("[poller] error unmarshalling order %s: %v", p.ReferenceNumber, err)
			continue
		}
		if err := ordering.WriteOrderXML(ticketReq, xmlInOrderDir); err != nil {
			log.Printf("[poller] error writing XML for order %s: %v", p.ReferenceNumber, err)
			go putOrderResult(cfg, p.ReferenceNumber, "failed", err.Error(), nil)
		} else {
			log.Printf("[poller] wrote XML for order ref=%s", p.ReferenceNumber)
			go reportOrderResult(cfg, p.ReferenceNumber, ticketReq, xmlDir, xmlCloseDir)
		}
	}
}

func reportOrderResult(cfg *config.Config, referenceNumber string, req ordering.CreateTicketRequest, xmlDir string, xmlCloseDir string) {
	deadline := time.Now().Add(pollerConfirmTimeout)
	for time.Now().Before(deadline) {
		conf, confFile, err := ordering.FindConfirmation(xmlDir, referenceNumber)
		if err == nil && conf != nil {
			if removeErr := os.Remove(confFile); removeErr != nil {
				log.Printf("[poller] warning: failed to remove confirmation file %s: %v", confFile, removeErr)
			}

			if conf.Transaction.ResponseCode != 0 {
				errText := ""
				if conf.Transaction.Error != nil {
					errText = conf.Transaction.Error.Text
				}
				putOrderResult(cfg, referenceNumber, "failed", errText, nil)
				return
			}

			var matchedTicket *positouch.Ticket
			tableNum, convErr := strconv.Atoi(req.TableNumber)
			if convErr != nil {
				log.Printf("[poller] warning: table_number '%s' is not a valid integer: %v", req.TableNumber, convErr)
			} else {
				tickets, tickErr := positouch.ReadAllTickets(xmlDir, xmlCloseDir)
				if tickErr != nil {
					log.Printf("[poller] warning: failed to read tickets for confirmation: %v", tickErr)
				} else {
					for i := range tickets {
						t := &tickets[i]
						if t.Table == tableNum && time.Since(t.OpenedAt) <= pollerTicketMatchWindow {
							matchedTicket = t
							break
						}
					}
				}
			}

			putOrderResult(cfg, referenceNumber, "created", "", matchedTicket)
			return
		}
		time.Sleep(pollerConfirmInterval)
	}

	putOrderResult(cfg, referenceNumber, "failed", fmt.Sprintf("timeout: no confirmation from POSitouch after %.0fs", pollerConfirmTimeout.Seconds()), nil)
}

func putOrderResult(cfg *config.Config, referenceNumber, status, errMsg string, ticket *positouch.Ticket) {
	locationID := cfg.LocationID
	if locationID == "" {
		locationID = cfg.Location.Name
	}

	base := strings.TrimRight(cfg.Cloud.Endpoint, "/")
	rawURL := fmt.Sprintf("%s/%s/tickets/%s/result", base, locationID, referenceNumber)

	body := struct {
		Status string            `json:"status"`
		Error  string            `json:"error,omitempty"`
		Ticket *positouch.Ticket `json:"ticket,omitempty"`
	}{
		Status: status,
		Error:  errMsg,
		Ticket: ticket,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Printf("[poller] error marshalling result for ref=%s: %v", referenceNumber, err)
		return
	}

	req, err := http.NewRequest("PUT", rawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("[poller] error building PUT request for ref=%s: %v", referenceNumber, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Cloud.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Cloud.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[poller] error sending result for ref=%s: %v", referenceNumber, err)
		return
	}
	defer resp.Body.Close()

	log.Printf("[poller] reported result for ref=%s status=%s http=%s", referenceNumber, status, resp.Status)
}