package cloud

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Client struct {
    Endpoint   string
    APIKey     string
    LocationID string
    HTTP       *http.Client
}

func NewClient(endpoint, apiKey, locationID string) *Client {
    return &Client{
        Endpoint:   endpoint,
        APIKey:     apiKey,
        LocationID: locationID,
        HTTP:       &http.Client{Timeout: 15 * time.Second},
    }
}

func (c *Client) SyncCache(data interface{}) error {
    body, err := json.Marshal(data)
    if err != nil {
        return err
    }
    req, err := http.NewRequest("PUT", c.Endpoint, bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Location-ID", c.LocationID)
    req.Header.Set("Authorization", "Bearer "+c.APIKey)

    resp, err := c.HTTP.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        return fmt.Errorf("server returned status %d", resp.StatusCode)
    }
    return nil
}