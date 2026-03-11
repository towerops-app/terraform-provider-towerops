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
	SiteID               *string `json:"site_id,omitempty"`
	OrganizationID       *string `json:"organization_id,omitempty"`
	Name                 *string `json:"name,omitempty"`
	IPAddress            string  `json:"ip_address"`
	Description          *string `json:"description,omitempty"`
	MonitoringEnabled    *bool   `json:"monitoring_enabled,omitempty"`
	SNMPEnabled          *bool   `json:"snmp_enabled,omitempty"`
	SNMPVersion          *string `json:"snmp_version,omitempty"`
	SNMPPort             *int    `json:"snmp_port,omitempty"`
	CheckIntervalSeconds *int    `json:"check_interval_seconds,omitempty"`
	// SNMPv3 fields
	SNMPv3SecurityLevel *string `json:"snmpv3_security_level,omitempty"`
	SNMPv3Username      *string `json:"snmpv3_username,omitempty"`
	SNMPv3AuthProtocol  *string `json:"snmpv3_auth_protocol,omitempty"`
	SNMPv3AuthPassword  *string `json:"snmpv3_auth_password,omitempty"`
	SNMPv3PrivProtocol  *string `json:"snmpv3_priv_protocol,omitempty"`
	SNMPv3PrivPassword  *string `json:"snmpv3_priv_password,omitempty"`
	InsertedAt          string  `json:"inserted_at,omitempty"`
}

// Organization represents a TowerOps organization.
type Organization struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name,omitempty"`
	Slug          string `json:"slug,omitempty"`
	UseSites      bool   `json:"use_sites"`
	SnmpCommunity string `json:"snmp_community,omitempty"`
}

// organizationResponse wraps the API response for organization endpoints.
type organizationResponse struct {
	Data Organization `json:"data"`
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

// OnCallSchedule represents a TowerOps on-call schedule.
type OnCallSchedule struct {
	ID          string  `json:"id,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Timezone    string  `json:"timezone"`
	InsertedAt  string  `json:"inserted_at,omitempty"`
}

// CreateSchedule creates a new on-call schedule.
func (c *Client) CreateSchedule(schedule OnCallSchedule) (*OnCallSchedule, error) {
	body := map[string]OnCallSchedule{"schedule": schedule}
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/schedules", body)
	if err != nil {
		return nil, err
	}

	var result OnCallSchedule
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetSchedule retrieves an on-call schedule by ID.
func (c *Client) GetSchedule(id string) (*OnCallSchedule, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/schedules/"+id, nil)
	if err != nil {
		return nil, err
	}

	var result OnCallSchedule
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// UpdateSchedule updates an existing on-call schedule.
func (c *Client) UpdateSchedule(id string, schedule OnCallSchedule) (*OnCallSchedule, error) {
	body := map[string]OnCallSchedule{"schedule": schedule}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/schedules/"+id, body)
	if err != nil {
		return nil, err
	}

	var result OnCallSchedule
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// DeleteSchedule deletes an on-call schedule.
func (c *Client) DeleteSchedule(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/schedules/"+id, nil)
	return err
}

// EscalationPolicyAPI represents a TowerOps escalation policy.
type EscalationPolicyAPI struct {
	ID          string  `json:"id,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	RepeatCount *int    `json:"repeat_count,omitempty"`
	InsertedAt  string  `json:"inserted_at,omitempty"`
}

// CreateEscalationPolicy creates a new escalation policy.
func (c *Client) CreateEscalationPolicy(policy EscalationPolicyAPI) (*EscalationPolicyAPI, error) {
	body := map[string]EscalationPolicyAPI{"escalation_policy": policy}
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/escalation_policies", body)
	if err != nil {
		return nil, err
	}

	var result EscalationPolicyAPI
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetEscalationPolicy retrieves an escalation policy by ID.
func (c *Client) GetEscalationPolicy(id string) (*EscalationPolicyAPI, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/escalation_policies/"+id, nil)
	if err != nil {
		return nil, err
	}

	var result EscalationPolicyAPI
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// UpdateEscalationPolicy updates an existing escalation policy.
func (c *Client) UpdateEscalationPolicy(id string, policy EscalationPolicyAPI) (*EscalationPolicyAPI, error) {
	body := map[string]EscalationPolicyAPI{"escalation_policy": policy}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/escalation_policies/"+id, body)
	if err != nil {
		return nil, err
	}

	var result EscalationPolicyAPI
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// DeleteEscalationPolicy deletes an escalation policy.
func (c *Client) DeleteEscalationPolicy(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/escalation_policies/"+id, nil)
	return err
}

// GetOrganization retrieves the current organization settings.
func (c *Client) GetOrganization() (*Organization, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/organization", nil)
	if err != nil {
		return nil, err
	}

	var result organizationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result.Data, nil
}

// UpdateOrganization updates the current organization settings.
func (c *Client) UpdateOrganization(org Organization) (*Organization, error) {
	body := map[string]Organization{"organization": org}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/organization", body)
	if err != nil {
		return nil, err
	}

	var result organizationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result.Data, nil
}
