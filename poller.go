package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/badpanda83/POSitouch-Integration/auth"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/driver"
	"github.com/badpanda83/POSitouch-Integration/entities"
	"github.com/badpanda83/POSitouch-Integration/ordering"
)

const (
	pollerConfirmTimeout  = 30 * time.Second
	pollerConfirmInterval = 2 * time.Second
)

func pollPendingOrders(cfg *config.Config, provider auth.TokenProvider, posDriver driver.POSDriver) {
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
	token, err := provider.GetAccessToken()
	if err != nil {
		log.Printf("[poller] error getting access token: %v", err)
		return
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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
		var ticketReq entities.CreateOrderRequest
		if err := json.Unmarshal(p.Payload, &ticketReq); err != nil {
			log.Printf("[poller] error unmarshalling order %s: %v", p.ReferenceNumber, err)
			continue
		}
		go reportOrderResult(cfg, provider, posDriver, p.ReferenceNumber, ticketReq)
	}
}

func reportOrderResult(cfg *config.Config, provider auth.TokenProvider, posDriver driver.POSDriver, referenceNumber string, req entities.CreateOrderRequest) {
	ticket, err := posDriver.CreateOrder(req)
	if err != nil {
		putOrderResult(cfg, provider, referenceNumber, "failed", err.Error(), nil)
		return
	}
	putOrderResult(cfg, provider, referenceNumber, "created", "", ticket)
}

func putOrderResult(cfg *config.Config, provider auth.TokenProvider, referenceNumber, status, errMsg string, ticket *entities.Ticket) {
	locationID := cfg.LocationID
	if locationID == "" {
		locationID = cfg.Location.Name
	}

	base := strings.TrimRight(cfg.Cloud.Endpoint, "/")
	rawURL := fmt.Sprintf("%s/%s/tickets/%s/result", base, locationID, referenceNumber)

	body := struct {
		Status string           `json:"status"`
		Error  string           `json:"error,omitempty"`
		Ticket *entities.Ticket `json:"ticket,omitempty"`
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
	token, err := provider.GetAccessToken()
	if err != nil {
		log.Printf("[poller] error getting access token for ref=%s: %v", referenceNumber, err)
		return
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[poller] error sending result for ref=%s: %v", referenceNumber, err)
		return
	}
	defer resp.Body.Close()

	log.Printf("[poller] reported result for ref=%s status=%s http=%s", referenceNumber, status, resp.Status)
}

func pollPendingPayments(cfg *config.Config, provider auth.TokenProvider, xmlInOrderDir string) {
	base := strings.TrimRight(cfg.Cloud.Endpoint, "/")
	locationID := cfg.LocationID
	if locationID == "" {
		locationID = cfg.Location.Name
	}

	rawURL := fmt.Sprintf("%s/%s/payments/pending", base, locationID)
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
	token, err := provider.GetAccessToken()
	if err != nil {
		log.Printf("[poller] error getting access token: %v", err)
		return
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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
	log.Printf("[poller] received %d pending payment(s)", len(pending))

	for _, p := range pending {
		var payReq ordering.PaymentRequest
		if err := json.Unmarshal(p.Payload, &payReq); err != nil {
			log.Printf("[poller] error unmarshalling payment %s: %v", p.ReferenceNumber, err)
			continue
		}
		if err := ordering.WritePaymentXML(payReq, xmlInOrderDir); err != nil {
			log.Printf("[poller] error writing payment XML for ref=%s: %v", payReq.ReferenceNumber, err)
			go putPaymentResult(cfg, provider, payReq.ReferenceNumber, "failed", err.Error())
		} else {
			log.Printf("[poller] wrote payment XML for ref=%s", payReq.ReferenceNumber)
			go reportPaymentResult(cfg, provider, payReq.ReferenceNumber, xmlInOrderDir)
		}
	}
}

func reportPaymentResult(cfg *config.Config, provider auth.TokenProvider, referenceNumber string, xmlDir string) {
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
				putPaymentResult(cfg, provider, referenceNumber, "failed", errText)
				return
			}

			putPaymentResult(cfg, provider, referenceNumber, "paid", "")
			return
		}
		time.Sleep(pollerConfirmInterval)
	}

	putPaymentResult(cfg, provider, referenceNumber, "failed", fmt.Sprintf("timeout: no confirmation from POSitouch after %.0fs", pollerConfirmTimeout.Seconds()))
}

func putPaymentResult(cfg *config.Config, provider auth.TokenProvider, referenceNumber, status, errMsg string) {
	locationID := cfg.LocationID
	if locationID == "" {
		locationID = cfg.Location.Name
	}

	base := strings.TrimRight(cfg.Cloud.Endpoint, "/")
	rawURL := fmt.Sprintf("%s/%s/payments/%s/result", base, locationID, referenceNumber)

	body := struct {
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}{
		Status: status,
		Error:  errMsg,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Printf("[poller] error marshalling payment result for ref=%s: %v", referenceNumber, err)
		return
	}

	req, err := http.NewRequest("PUT", rawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("[poller] error building PUT request for ref=%s: %v", referenceNumber, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	token, err := provider.GetAccessToken()
	if err != nil {
		log.Printf("[poller] error getting access token for ref=%s: %v", referenceNumber, err)
		return
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[poller] error sending payment result for ref=%s: %v", referenceNumber, err)
		return
	}
	defer resp.Body.Close()

	log.Printf("[poller] reported payment result for ref=%s status=%s http=%s", referenceNumber, status, resp.Status)
}
