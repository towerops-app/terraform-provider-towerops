package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
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
//
// SnmpCommunity is write-only: the API accepts it on create and update but
// never returns it, reporting SNMPCommunitySet instead.
type Site struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name"`
	Description      *string  `json:"description,omitempty"`
	Location         *string  `json:"location,omitempty"`
	Address          *string  `json:"address,omitempty"`
	Latitude         *float64 `json:"latitude,omitempty"`
	Longitude        *float64 `json:"longitude,omitempty"`
	DisplayOrder     *int     `json:"display_order,omitempty"`
	SNMPCommunity    *string  `json:"snmp_community,omitempty"`
	SNMPCommunitySet bool     `json:"snmp_community_set,omitempty"`
	SNMPVersion      *string  `json:"snmp_version,omitempty"`
	SNMPPort         *int     `json:"snmp_port,omitempty"`
	SNMPTransport    *string  `json:"snmp_transport,omitempty"`
	AgentTokenID     *string  `json:"agent_token_id,omitempty"`
	ParentSiteID     *string  `json:"parent_site_id,omitempty"`
	InsertedAt       string   `json:"inserted_at,omitempty"`
}

// Device represents a TowerOps device.
//
// The SNMPv3 passwords are write-only; the API reports whether each is set
// through SNMPv3AuthPasswordSet and SNMPv3PrivPasswordSet.
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
	DeviceRole           *string `json:"device_role,omitempty"`
	EscalationPolicyID   *string `json:"escalation_policy_id,omitempty"`
	// SNMPv3 fields
	SNMPv3SecurityLevel   *string `json:"snmpv3_security_level,omitempty"`
	SNMPv3Username        *string `json:"snmpv3_username,omitempty"`
	SNMPv3AuthProtocol    *string `json:"snmpv3_auth_protocol,omitempty"`
	SNMPv3AuthPassword    *string `json:"snmpv3_auth_password,omitempty"`
	SNMPv3AuthPasswordSet bool    `json:"snmpv3_auth_password_set,omitempty"`
	SNMPv3PrivProtocol    *string `json:"snmpv3_priv_protocol,omitempty"`
	SNMPv3PrivPassword    *string `json:"snmpv3_priv_password,omitempty"`
	SNMPv3PrivPasswordSet bool    `json:"snmpv3_priv_password_set,omitempty"`
	InsertedAt            string  `json:"inserted_at,omitempty"`
}

// Organization represents a TowerOps organization.
//
// SnmpCommunity and the SNMPv3/MikroTik passwords are write-only. The API
// reports only SNMPCommunitySet on read.
type Organization struct {
	ID                  string  `json:"id,omitempty"`
	Name                string  `json:"name,omitempty"`
	Slug                string  `json:"slug,omitempty"`
	SubscriptionPlan    string  `json:"subscription_plan,omitempty"`
	UseSites            bool    `json:"use_sites"`
	SnmpCommunity       string  `json:"snmp_community,omitempty"`
	SNMPCommunitySet    bool    `json:"snmp_community_set,omitempty"`
	SNMPVersion         *string `json:"snmp_version,omitempty"`
	SNMPPort            *int    `json:"snmp_port,omitempty"`
	SNMPTransport       *string `json:"snmp_transport,omitempty"`
	SNMPv3SecurityLevel *string `json:"snmpv3_security_level,omitempty"`
	SNMPv3Username      *string `json:"snmpv3_username,omitempty"`
	SNMPv3AuthProtocol  *string `json:"snmpv3_auth_protocol,omitempty"`
	SNMPv3AuthPassword  *string `json:"snmpv3_auth_password,omitempty"`
	SNMPv3PrivProtocol  *string `json:"snmpv3_priv_protocol,omitempty"`
	SNMPv3PrivPassword  *string `json:"snmpv3_priv_password,omitempty"`
	MikrotikEnabled     *bool   `json:"mikrotik_enabled,omitempty"`
	MikrotikUsername    *string `json:"mikrotik_username,omitempty"`
	MikrotikPassword    *string `json:"mikrotik_password,omitempty"`
	MikrotikPort        *int    `json:"mikrotik_port,omitempty"`
	MikrotikSSHPort     *int    `json:"mikrotik_ssh_port,omitempty"`
	MikrotikUseSSL      *bool   `json:"mikrotik_use_ssl,omitempty"`
	DefaultAgentTokenID *string `json:"default_agent_token_id,omitempty"`
	AlertRouting        *string `json:"alert_routing,omitempty"`
	InsertedAt          string  `json:"inserted_at,omitempty"`
	UpdatedAt           string  `json:"updated_at,omitempty"`
}

// APIError is the /api/v1 error envelope:
//
//	{"error": {"code": "...", "message": "...", "details": {"fields": {...}}}}
//
// Code is one of the stable machine-readable codes the API publishes:
// authentication_required, invalid_token, forbidden, insufficient_scope,
// not_found, bad_request, validation_error, conflict, upstream_error,
// timeout, rate_limited, internal_error.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Fields     map[string][]string
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "API error (%d", e.StatusCode)
	if e.Code != "" {
		fmt.Fprintf(&b, " %s", e.Code)
	}
	b.WriteString("): ")
	b.WriteString(e.Message)

	if len(e.Fields) > 0 {
		names := make([]string, 0, len(e.Fields))
		for name := range e.Fields {
			names = append(names, name)
		}
		sort.Strings(names)

		b.WriteString(" (")
		for i, name := range names {
			if i > 0 {
				b.WriteString("; ")
			}
			fmt.Fprintf(&b, "%s: %s", name, strings.Join(e.Fields[name], ", "))
		}
		b.WriteString(")")
	}

	return b.String()
}

// errorEnvelope mirrors the wire format of an /api/v1 failure.
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details struct {
			Fields map[string][]string `json:"fields"`
		} `json:"details"`
	} `json:"error"`
}

// parseAPIError turns a failure body into an *APIError. A body that does not
// carry the envelope still produces an error rather than being swallowed: the
// raw payload becomes the message so the operator sees something actionable.
func parseAPIError(status int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: status}

	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err == nil && env.Error.Message != "" {
		apiErr.Code = env.Error.Code
		apiErr.Message = env.Error.Message
		apiErr.Fields = env.Error.Details.Fields
		return apiErr
	}

	apiErr.Message = strings.TrimSpace(string(body))
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(status)
	}
	return apiErr
}

// decodeData unwraps the `{"data": ...}` success envelope every /api/v1
// endpoint returns.
func decodeData[T any](body []byte) (*T, error) {
	var envelope struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &envelope.Data, nil
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
		apiErr := parseAPIError(resp.StatusCode, respBody)
		if resp.StatusCode == http.StatusNotFound || apiErr.Code == "not_found" {
			return nil, ErrNotFound
		}
		return nil, apiErr
	}

	return respBody, nil
}

// CreateSite creates a new site.
func (c *Client) CreateSite(site Site) (*Site, error) {
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/sites", map[string]Site{"site": site})
	if err != nil {
		return nil, err
	}
	return decodeData[Site](respBody)
}

// GetSite retrieves a site by ID.
func (c *Client) GetSite(id string) (*Site, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/sites/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[Site](respBody)
}

// UpdateSite updates an existing site.
func (c *Client) UpdateSite(id string, site Site) (*Site, error) {
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/sites/"+id, map[string]Site{"site": site})
	if err != nil {
		return nil, err
	}
	return decodeData[Site](respBody)
}

// DeleteSite deletes a site.
func (c *Client) DeleteSite(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/sites/"+id, nil)
	return err
}

// CreateDevice creates a new device.
func (c *Client) CreateDevice(device Device) (*Device, error) {
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/devices", map[string]Device{"device": device})
	if err != nil {
		return nil, err
	}
	return decodeData[Device](respBody)
}

// GetDevice retrieves a device by ID.
func (c *Client) GetDevice(id string) (*Device, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/devices/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[Device](respBody)
}

// UpdateDevice updates an existing device.
func (c *Client) UpdateDevice(id string, device Device) (*Device, error) {
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/devices/"+id, map[string]Device{"device": device})
	if err != nil {
		return nil, err
	}
	return decodeData[Device](respBody)
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
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/schedules", map[string]OnCallSchedule{"schedule": schedule})
	if err != nil {
		return nil, err
	}
	return decodeData[OnCallSchedule](respBody)
}

// GetSchedule retrieves an on-call schedule by ID.
func (c *Client) GetSchedule(id string) (*OnCallSchedule, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/schedules/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[OnCallSchedule](respBody)
}

// UpdateSchedule updates an existing on-call schedule.
func (c *Client) UpdateSchedule(id string, schedule OnCallSchedule) (*OnCallSchedule, error) {
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/schedules/"+id, map[string]OnCallSchedule{"schedule": schedule})
	if err != nil {
		return nil, err
	}
	return decodeData[OnCallSchedule](respBody)
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
	return decodeData[EscalationPolicyAPI](respBody)
}

// GetEscalationPolicy retrieves an escalation policy by ID.
func (c *Client) GetEscalationPolicy(id string) (*EscalationPolicyAPI, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/escalation_policies/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[EscalationPolicyAPI](respBody)
}

// UpdateEscalationPolicy updates an existing escalation policy.
func (c *Client) UpdateEscalationPolicy(id string, policy EscalationPolicyAPI) (*EscalationPolicyAPI, error) {
	body := map[string]EscalationPolicyAPI{"escalation_policy": policy}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/escalation_policies/"+id, body)
	if err != nil {
		return nil, err
	}
	return decodeData[EscalationPolicyAPI](respBody)
}

// DeleteEscalationPolicy deletes an escalation policy.
func (c *Client) DeleteEscalationPolicy(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/escalation_policies/"+id, nil)
	return err
}

// Agent represents a TowerOps agent token.
//
// Token is only ever populated on the create response; the API never returns
// it again.
type Agent struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name"`
	Enabled     *bool          `json:"enabled,omitempty"`
	LastSeenAt  *string        `json:"last_seen_at,omitempty"`
	LastIP      *string        `json:"last_ip,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	DeviceCount *int           `json:"device_count,omitempty"`
	InsertedAt  string         `json:"inserted_at,omitempty"`
	Token       string         `json:"token,omitempty"`
}

// CreateAgent creates a new agent token.
//
// The name rides at the top level of the body, not under an "agent" key: the
// controller matches on `%{"name" => name}` and answers anything else with a
// 400 bad_request.
func (c *Client) CreateAgent(agent Agent) (*Agent, error) {
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/agents", map[string]string{"name": agent.Name})
	if err != nil {
		return nil, err
	}
	return decodeData[Agent](respBody)
}

// GetAgent retrieves an agent by ID.
func (c *Client) GetAgent(id string) (*Agent, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/agents/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[Agent](respBody)
}

// DeleteAgent deletes an agent.
func (c *Client) DeleteAgent(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/agents/"+id, nil)
	return err
}

// Integration represents a TowerOps integration.
type Integration struct {
	ID                  string  `json:"id,omitempty"`
	Provider            string  `json:"provider"`
	Enabled             *bool   `json:"enabled,omitempty"`
	SyncIntervalMinutes *int    `json:"sync_interval_minutes,omitempty"`
	LastSyncedAt        *string `json:"last_synced_at,omitempty"`
	LastSyncStatus      *string `json:"last_sync_status,omitempty"`
	LastSyncMessage     *string `json:"last_sync_message,omitempty"`
	InsertedAt          string  `json:"inserted_at,omitempty"`
	UpdatedAt           string  `json:"updated_at,omitempty"`
}

// integrationRequest is the create and update body for an integration.
//
// The controller's attribute allowlist is provider, enabled, config, api_key,
// api_url and sync_interval_minutes, and its changeset then casts only
// provider, enabled, settings and sync_interval_minutes. Those three fields
// below are therefore the entire settable surface over REST today.
type integrationRequest struct {
	Provider            string `json:"provider"`
	Enabled             *bool  `json:"enabled,omitempty"`
	SyncIntervalMinutes *int   `json:"sync_interval_minutes,omitempty"`
}

// CreateIntegration creates a new integration.
func (c *Client) CreateIntegration(integration integrationRequest) (*Integration, error) {
	body := map[string]integrationRequest{"integration": integration}
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/integrations", body)
	if err != nil {
		return nil, err
	}
	return decodeData[Integration](respBody)
}

// GetIntegration retrieves an integration by ID.
func (c *Client) GetIntegration(id string) (*Integration, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/integrations/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[Integration](respBody)
}

// UpdateIntegration updates an existing integration.
func (c *Client) UpdateIntegration(id string, integration integrationRequest) (*Integration, error) {
	body := map[string]integrationRequest{"integration": integration}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/integrations/"+id, body)
	if err != nil {
		return nil, err
	}
	return decodeData[Integration](respBody)
}

// DeleteIntegration deletes an integration.
func (c *Client) DeleteIntegration(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/integrations/"+id, nil)
	return err
}

// MaintenanceWindowAPI represents a TowerOps maintenance window.
type MaintenanceWindowAPI struct {
	ID             string  `json:"id,omitempty"`
	Name           string  `json:"name"`
	Reason         *string `json:"reason,omitempty"`
	StartsAt       string  `json:"starts_at"`
	EndsAt         string  `json:"ends_at"`
	Recurring      *bool   `json:"recurring,omitempty"`
	RecurrenceRule *string `json:"recurrence_rule,omitempty"`
	SuppressAlerts *bool   `json:"suppress_alerts,omitempty"`
	SiteID         *string `json:"site_id,omitempty"`
	DeviceID       *string `json:"device_id,omitempty"`
	InsertedAt     string  `json:"inserted_at,omitempty"`
}

// CreateMaintenanceWindow creates a new maintenance window.
func (c *Client) CreateMaintenanceWindow(window MaintenanceWindowAPI) (*MaintenanceWindowAPI, error) {
	body := map[string]MaintenanceWindowAPI{"maintenance_window": window}
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/maintenance_windows", body)
	if err != nil {
		return nil, err
	}
	return decodeData[MaintenanceWindowAPI](respBody)
}

// GetMaintenanceWindow retrieves a maintenance window by ID.
func (c *Client) GetMaintenanceWindow(id string) (*MaintenanceWindowAPI, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/maintenance_windows/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[MaintenanceWindowAPI](respBody)
}

// UpdateMaintenanceWindow updates an existing maintenance window.
func (c *Client) UpdateMaintenanceWindow(id string, window MaintenanceWindowAPI) (*MaintenanceWindowAPI, error) {
	body := map[string]MaintenanceWindowAPI{"maintenance_window": window}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/maintenance_windows/"+id, body)
	if err != nil {
		return nil, err
	}
	return decodeData[MaintenanceWindowAPI](respBody)
}

// DeleteMaintenanceWindow deletes a maintenance window.
func (c *Client) DeleteMaintenanceWindow(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/maintenance_windows/"+id, nil)
	return err
}

// Check represents a TowerOps service check.
type Check struct {
	ID                   string         `json:"id,omitempty"`
	Name                 string         `json:"name"`
	CheckType            string         `json:"check_type"`
	Description          *string        `json:"description,omitempty"`
	Enabled              *bool          `json:"enabled,omitempty"`
	Alerting             *bool          `json:"alerting,omitempty"`
	DeviceID             *string        `json:"device_id,omitempty"`
	AgentTokenID         *string        `json:"agent_token_id,omitempty"`
	IntervalSeconds      *int           `json:"interval_seconds,omitempty"`
	TimeoutMs            *int           `json:"timeout_ms,omitempty"`
	RetryIntervalSeconds *int           `json:"retry_interval_seconds,omitempty"`
	MaxCheckAttempts     *int           `json:"max_check_attempts,omitempty"`
	Config               map[string]any `json:"config"`
	CurrentState         *int           `json:"current_state,omitempty"`
	CurrentStateType     *string        `json:"current_state_type,omitempty"`
	LastCheckAt          *string        `json:"last_check_at,omitempty"`
	InsertedAt           string         `json:"inserted_at,omitempty"`
}

// CreateCheck creates a new service check.
func (c *Client) CreateCheck(check Check) (*Check, error) {
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/checks", map[string]Check{"check": check})
	if err != nil {
		return nil, err
	}
	return decodeData[Check](respBody)
}

// GetCheck retrieves a check by ID.
func (c *Client) GetCheck(id string) (*Check, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/checks/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[Check](respBody)
}

// UpdateCheck updates an existing check.
func (c *Client) UpdateCheck(id string, check Check) (*Check, error) {
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/checks/"+id, map[string]Check{"check": check})
	if err != nil {
		return nil, err
	}
	return decodeData[Check](respBody)
}

// DeleteCheck deletes a check.
func (c *Client) DeleteCheck(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/checks/"+id, nil)
	return err
}

// CoverageBBox is the geographic extent of a computed coverage heatmap.
type CoverageBBox struct {
	MinLat *float64 `json:"min_lat"`
	MaxLat *float64 `json:"max_lat"`
	MinLon *float64 `json:"min_lon"`
	MaxLon *float64 `json:"max_lon"`
}

// Coverage represents an RF coverage prediction for a single antenna.
//
// Only SI inputs are sent. The API also accepts imperial virtual fields
// (height_agl_ft, radius_mi, frequency_ghz and friends) but converts them to
// SI before persisting, so a request in imperial units always reads back in
// metric and would make every plan permanently dirty.
type Coverage struct {
	ID                  string   `json:"id,omitempty"`
	Name                string   `json:"name"`
	OrganizationID      string   `json:"organization_id,omitempty"`
	SiteID              string   `json:"site_id,omitempty"`
	DeviceID            *string  `json:"device_id,omitempty"`
	AntennaSlug         string   `json:"antenna_slug"`
	FrequencyMHz        *int     `json:"frequency_mhz,omitempty"`
	TxPowerDBm          *float64 `json:"tx_power_dbm,omitempty"`
	CableLossDB         *float64 `json:"cable_loss_db,omitempty"`
	SmGainDBi           *float64 `json:"sm_gain_dbi,omitempty"`
	HeightAGLM          *float64 `json:"height_agl_m,omitempty"`
	HeightAboveRooftopM *float64 `json:"height_above_rooftop_m,omitempty"`
	TxClearanceM        *float64 `json:"tx_clearance_m,omitempty"`
	AzimuthDeg          *float64 `json:"azimuth_deg,omitempty"`
	DowntiltDeg         *float64 `json:"downtilt_deg,omitempty"`
	RadiusM             *int     `json:"radius_m,omitempty"`
	CellSizeM           *int     `json:"cell_size_m,omitempty"`
	ReceiverHeightM     *float64 `json:"receiver_height_m,omitempty"`
	RxThresholdDBm      *float64 `json:"rx_threshold_dbm,omitempty"`
	FoliageTuning       *int     `json:"foliage_tuning,omitempty"`
	LatitudeOverride    *float64 `json:"latitude_override,omitempty"`
	LongitudeOverride   *float64 `json:"longitude_override,omitempty"`

	// Computed by the coverage worker.
	Status       string        `json:"status,omitempty"`
	ProgressPct  *int          `json:"progress_pct,omitempty"`
	ErrorMessage *string       `json:"error_message,omitempty"`
	ComputedAt   *string       `json:"computed_at,omitempty"`
	PNGPath      *string       `json:"png_path,omitempty"`
	RasterPath   *string       `json:"raster_path,omitempty"`
	BBox         *CoverageBBox `json:"bbox,omitempty"`
	InsertedAt   string        `json:"inserted_at,omitempty"`
	UpdatedAt    string        `json:"updated_at,omitempty"`
}

// CreateCoverage creates a coverage and enqueues its first compute.
func (c *Client) CreateCoverage(coverage Coverage) (*Coverage, error) {
	body := map[string]Coverage{"coverage": coverage}
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/coverages", body)
	if err != nil {
		return nil, err
	}
	return decodeData[Coverage](respBody)
}

// GetCoverage retrieves a coverage by ID.
func (c *Client) GetCoverage(id string) (*Coverage, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/coverages/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[Coverage](respBody)
}

// UpdateCoverage updates a coverage. It does not recompute the heatmap; call
// RecomputeCoverage for that.
func (c *Client) UpdateCoverage(id string, coverage Coverage) (*Coverage, error) {
	body := map[string]Coverage{"coverage": coverage}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/coverages/"+id, body)
	if err != nil {
		return nil, err
	}
	return decodeData[Coverage](respBody)
}

// RecomputeCoverage re-runs the compute pipeline for an existing coverage.
func (c *Client) RecomputeCoverage(id string) (*Coverage, error) {
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/coverages/"+id+"/recompute", nil)
	if err != nil {
		return nil, err
	}
	return decodeData[Coverage](respBody)
}

// DeleteCoverage deletes a coverage.
func (c *Client) DeleteCoverage(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/coverages/"+id, nil)
	return err
}

// WebhookEndpoint represents an outbound webhook subscription.
//
// Secret is write-only: the API returns it exactly once, on the create
// response, and omits it from every later read.
type WebhookEndpoint struct {
	ID                  string   `json:"id,omitempty"`
	Name                *string  `json:"name,omitempty"`
	URL                 string   `json:"url"`
	Secret              *string  `json:"secret,omitempty"`
	Events              []string `json:"events"`
	Active              *bool    `json:"active,omitempty"`
	ConsecutiveFailures *int     `json:"consecutive_failures,omitempty"`
	LastDeliveryAt      *string  `json:"last_delivery_at,omitempty"`
	LastDeliveryStatus  *string  `json:"last_delivery_status,omitempty"`
	CreatedAt           string   `json:"created_at,omitempty"`
}

// CreateWebhookEndpoint registers an outbound webhook endpoint.
func (c *Client) CreateWebhookEndpoint(endpoint WebhookEndpoint) (*WebhookEndpoint, error) {
	body := map[string]WebhookEndpoint{"webhook_endpoint": endpoint}
	respBody, err := c.doRequest(http.MethodPost, "/api/v1/webhook_endpoints", body)
	if err != nil {
		return nil, err
	}
	return decodeData[WebhookEndpoint](respBody)
}

// GetWebhookEndpoint retrieves a webhook endpoint by ID.
func (c *Client) GetWebhookEndpoint(id string) (*WebhookEndpoint, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/webhook_endpoints/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeData[WebhookEndpoint](respBody)
}

// UpdateWebhookEndpoint updates a webhook endpoint.
func (c *Client) UpdateWebhookEndpoint(id string, endpoint WebhookEndpoint) (*WebhookEndpoint, error) {
	body := map[string]WebhookEndpoint{"webhook_endpoint": endpoint}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/webhook_endpoints/"+id, body)
	if err != nil {
		return nil, err
	}
	return decodeData[WebhookEndpoint](respBody)
}

// DeleteWebhookEndpoint deletes a webhook endpoint.
func (c *Client) DeleteWebhookEndpoint(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v1/webhook_endpoints/"+id, nil)
	return err
}

// GetOrganization retrieves the current organization settings.
func (c *Client) GetOrganization() (*Organization, error) {
	respBody, err := c.doRequest(http.MethodGet, "/api/v1/organization", nil)
	if err != nil {
		return nil, err
	}
	return decodeData[Organization](respBody)
}

// UpdateOrganization updates the current organization settings.
func (c *Client) UpdateOrganization(org Organization) (*Organization, error) {
	body := map[string]Organization{"organization": org}
	respBody, err := c.doRequest(http.MethodPatch, "/api/v1/organization", body)
	if err != nil {
		return nil, err
	}
	return decodeData[Organization](respBody)
}
