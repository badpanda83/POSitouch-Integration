package cloud

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"time"
)

type Client struct {
	Endpoint   string // Base API URL, e.g. https://your-cloud/api/v1/pos-data
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

// Sync a specific cache file, sending its contents to the corresponding cloud endpoint
func (c *Client) SyncEntityCache(entity, cacheDir string) error {
	file := filepath.Join(cacheDir, entity+".cache")
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read cache file: %w", err)
	}

	// Your cloud expects the body to be a JSON array (not a wrapped struct)
	url := fmt.Sprintf("%s/%s/%s", c.Endpoint, c.LocationID, entity) // e.g. https://.../api/v1/pos-data/Smittys/employees
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

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

// For convenience, sync all entities in a loop:
func (c *Client) SyncAllEntities(cacheDir string, entities []string) error {
	for _, entity := range entities {
		if err := c.SyncEntityCache(entity, cacheDir); err != nil {
			return fmt.Errorf("sync entity %s: %w", entity, err)
		}
	}
	return nil
}