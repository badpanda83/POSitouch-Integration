package micros3700driver

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/config"
)

const microsNS = "http://www.micros.com/res/pos/webservices/general/v1"

// soapResponseEnvelope is used to unmarshal a SOAP 1.1 response.
type soapResponseEnvelope struct {
	XMLName xml.Name         `xml:"Envelope"`
	Body    soapResponseBody `xml:"Body"`
}

type soapResponseBody struct {
	Inner []byte `xml:",innerxml"`
}

// postSOAP wraps payload in a SOAP 1.1 envelope and POSTs it to the MICROS
// Transaction Services endpoint. It sets the SOAPAction header, uses HTTP Basic
// Auth when configured, and returns the inner contents of <soap:Body> so callers
// do not need to parse the envelope themselves.
func postSOAP(cfg *config.MICROS3700Config, soapAction string, payload interface{}) ([]byte, error) {
	payloadBytes, err := xml.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("micros3700: marshal SOAP payload: %w", err)
	}

	body := fmt.Sprintf(
		`<?xml version="1.0" encoding="utf-8"?>`+
			`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">`+
			`<soap:Body>%s</soap:Body>`+
			`</soap:Envelope>`,
		string(payloadBytes),
	)

	req, err := http.NewRequest(http.MethodPost, cfg.TransactionServicesURL, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("micros3700: create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", soapAction)
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

	// Extract the inner contents of <soap:Body>.
	var envelope soapResponseEnvelope
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("micros3700: unmarshal SOAP response envelope: %w", err)
	}
	return envelope.Body.Inner, nil
}
