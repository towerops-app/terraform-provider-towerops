package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrNotFound is returned when a resource is not found (404).
var ErrNotFound = errors.New("resource not found")

const defaultBaseURL = "https://towerops.net"

// Client is the TowerOps API client.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a new TowerOps API client.
func NewClient(token, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Site represents a TowerOps site.
type Site struct {
	ID            string  `json:"id,omitempty"`
	Name          string  `json:"name"`
	Location      *string `json:"location,omitempty"`
	SNMPCommunity *string `json:"snmp_community,omitempty"`
	InsertedAt    string  `json:"inserted_at,omitempty"`
}

// Device represents a TowerOps device.
type Device struct {
	ID                   string  `json:"id,omitempty"`
	SiteID               string  `json:"site_id"`
	Name                 *string `json:"name,omitempty"`
	IPAddress            string  `json:"ip_address"`
	Description          *string `json:"description,omitempty"`
	MonitoringEnabled    *bool   `json:"monitoring_enabled,omitempty"`
	SNMPEnabled          *bool   `json:"snmp_enabled,omitempty"`
	SNMPVersion          *string `json:"snmp_version,omitempty"`
	SNMPPort             *int    `json:"snmp_port,omitempty"`
	CheckIntervalSeconds *int    `json:"check_interval_seconds,omitempty"`
	InsertedAt           string  `json:"inserted_at,omitempty"`
}

// APIError represents an error response from the API.
type APIError struct {
	Error  string            `json:"error,omitempty"`
	Errors map[string]string `json:"errors,omitempty"`
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			if apiErr.Error != "" {
				return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Error)
			}
			if len(apiErr.Errors) > 0 {
				return nil, fmt.Errorf("API validation error (%d): %v", resp.StatusCode, apiErr.Errors)
			}
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// CreateSite creates a new site.
func (c *Client) CreateSite(site Site) (*Site, error) {
	body := map[string]Site{"site": site}
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/sites", body)
	if err != nil {
		return nil, err
	}

	var result Site
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetSite retrieves a site by ID.
func (c *Client) GetSite(id string) (*Site, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/sites/"+id, nil)
	if err != nil {
		return nil, err
	}

	var result Site
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// UpdateSite updates an existing site.
func (c *Client) UpdateSite(id string, site Site) (*Site, error) {
	body := map[string]Site{"site": site}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/sites/"+id, body)
	if err != nil {
		return nil, err
	}

	var result Site
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// DeleteSite deletes a site.
func (c *Client) DeleteSite(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/sites/"+id, nil)
	return err
}

// CreateDevice creates a new device.
func (c *Client) CreateDevice(device Device) (*Device, error) {
	body := map[string]Device{"device": device}
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/devices", body)
	if err != nil {
		return nil, err
	}

	var result Device
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetDevice retrieves a device by ID.
func (c *Client) GetDevice(id string) (*Device, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/devices/"+id, nil)
	if err != nil {
		return nil, err
	}

	var result Device
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// UpdateDevice updates an existing device.
func (c *Client) UpdateDevice(id string, device Device) (*Device, error) {
	body := map[string]Device{"device": device}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/devices/"+id, body)
	if err != nil {
		return nil, err
	}

	var result Device
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// DeleteDevice deletes a device.
func (c *Client) DeleteDevice(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/devices/"+id, nil)
	return err
}
