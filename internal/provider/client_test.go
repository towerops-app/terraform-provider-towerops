package provider

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_ErrNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"code": "not_found", "message": "Resource not found"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.GetDevice("nonexistent-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// The API also answers not_found on a few non-404 statuses. The code, not the
// status alone, decides whether the resource is gone.
func TestClient_ErrNotFoundFromErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"code": "not_found", "message": "Site not found"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.GetSite("nonexistent-id")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestClient_GetDevice_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/devices/device-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"id": "device-123",
				"site_id": "site-456",
				"ip_address": "192.168.1.1",
				"name": "Test Device",
				"inserted_at": "2024-01-01T00:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device, err := client.GetDevice("device-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if device.ID != "device-123" {
		t.Errorf("expected ID device-123, got %s", device.ID)
	}
	if device.SiteID == nil || *device.SiteID != "site-456" {
		t.Errorf("expected SiteID site-456, got %v", device.SiteID)
	}
	if device.IPAddress != "192.168.1.1" {
		t.Errorf("expected IPAddress 192.168.1.1, got %s", device.IPAddress)
	}
}

func TestClient_CreateDevice_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/devices" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"data": {
				"id": "new-device-id",
				"site_id": "site-456",
				"ip_address": "192.168.1.100",
				"name": "New Device",
				"monitoring_enabled": true,
				"snmp_enabled": true,
				"inserted_at": "2024-01-01T00:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{
		SiteID:    new("site-456"),
		IPAddress: "192.168.1.100",
	}

	created, err := client.CreateDevice(device)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created.ID != "new-device-id" {
		t.Errorf("expected ID new-device-id, got %s", created.ID)
	}
}

func TestClient_UpdateDevice_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/devices/device-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"id": "device-123",
				"site_id": "site-456",
				"ip_address": "192.168.1.200",
				"name": "Updated Device",
				"inserted_at": "2024-01-01T00:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{
		SiteID:    new("site-456"),
		IPAddress: "192.168.1.200",
	}

	updated, err := client.UpdateDevice("device-123", device)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.IPAddress != "192.168.1.200" {
		t.Errorf("expected IPAddress 192.168.1.200, got %s", updated.IPAddress)
	}
}

func TestClient_UpdateDevice_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"code": "not_found", "message": "Device not found"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{
		SiteID:    new("site-456"),
		IPAddress: "192.168.1.200",
	}

	_, err := client.UpdateDevice("nonexistent", device)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestClient_DeleteDevice_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/devices/device-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	err := client.DeleteDevice("device-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"code": "bad_request", "message": "Invalid request"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.GetDevice("device-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should not be ErrNotFound for 400 errors
	if errors.Is(err, ErrNotFound) {
		t.Error("did not expect ErrNotFound for 400 error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
	if apiErr.Code != "bad_request" {
		t.Errorf("expected code bad_request, got %q", apiErr.Code)
	}
	if got, want := err.Error(), "API error (400 bad_request): Invalid request"; got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestClient_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{
			"error": {
				"code": "validation_error",
				"message": "Validation failed",
				"details": {"fields": {"ip_address": ["is invalid"]}}
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{
		SiteID:    new("site-456"),
		IPAddress: "invalid",
	}

	_, err := client.CreateDevice(device)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if errors.Is(err, ErrNotFound) {
		t.Error("did not expect ErrNotFound for validation error")
	}

	want := "API error (422 validation_error): Validation failed (ip_address: is invalid)"
	if got := err.Error(); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

// A 422 carries per-field messages. The rendering has to name every field,
// in a stable order, so a practitioner can act on the diagnostic.
func TestAPIError_RendersFieldsSorted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{
			"error": {
				"code": "validation_error",
				"message": "Validation failed",
				"details": {
					"fields": {
						"name": ["can't be blank", "is too short"],
						"alert_routing": ["must be builtin, pagerduty, or both"]
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.CreateSite(Site{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := "API error (422 validation_error): Validation failed " +
		"(alert_routing: must be builtin, pagerduty, or both; name: can't be blank, is too short)"
	if got := err.Error(); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

// A failure body that is not the envelope (a proxy error page, say) must not
// be swallowed: the raw payload has to reach the operator.
func TestAPIError_NonEnvelopeBodyKeepsRawPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`<html><body>502 Bad Gateway</body></html>`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.GetSite("site-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", apiErr.StatusCode)
	}
	if apiErr.Code != "" {
		t.Errorf("expected empty code, got %q", apiErr.Code)
	}
	if !strings.Contains(err.Error(), "502 Bad Gateway") {
		t.Errorf("expected raw payload in error, got %q", err.Error())
	}
}

func TestClient_GetSite_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/sites/site-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"id": "site-123",
				"name": "Test Site",
				"location": "New York",
				"inserted_at": "2024-01-01T00:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	site, err := client.GetSite("site-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if site.ID != "site-123" {
		t.Errorf("expected ID site-123, got %s", site.ID)
	}
	if site.Name != "Test Site" {
		t.Errorf("expected Name 'Test Site', got %s", site.Name)
	}
}

func TestClient_GetSite_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"code": "not_found", "message": "Site not found"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.GetSite("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestNewClient_DefaultBaseURL(t *testing.T) {
	client := NewClient("test-token", "")
	if client.BaseURL != defaultBaseURL {
		t.Errorf("expected default base URL %s, got %s", defaultBaseURL, client.BaseURL)
	}
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	client := NewClient("test-token", "https://custom.example.com")
	if client.BaseURL != "https://custom.example.com" {
		t.Errorf("expected custom base URL, got %s", client.BaseURL)
	}
}

func TestClient_CreateSite_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/sites" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"data": {
				"id": "new-site-id",
				"name": "New Site",
				"location": "Boston",
				"inserted_at": "2024-01-01T00:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	site := Site{
		Name: "New Site",
	}
	location := "Boston"
	site.Location = &location

	created, err := client.CreateSite(site)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created.ID != "new-site-id" {
		t.Errorf("expected ID new-site-id, got %s", created.ID)
	}
	if created.Name != "New Site" {
		t.Errorf("expected Name 'New Site', got %s", created.Name)
	}
}

func TestClient_CreateSite_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"code": "bad_request", "message": "Missing 'site' parameter"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	site := Site{}
	_, err := client.CreateSite(site)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_CreateSite_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	site := Site{Name: "Test"}
	_, err := client.CreateSite(site)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestClient_UpdateSite_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/sites/site-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"id": "site-123",
				"name": "Updated Site",
				"location": "Chicago",
				"inserted_at": "2024-01-01T00:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	site := Site{
		Name: "Updated Site",
	}

	updated, err := client.UpdateSite("site-123", site)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Name != "Updated Site" {
		t.Errorf("expected Name 'Updated Site', got %s", updated.Name)
	}
}

func TestClient_UpdateSite_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"code": "not_found", "message": "Site not found"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	site := Site{Name: "Test"}
	_, err := client.UpdateSite("nonexistent", site)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestClient_UpdateSite_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	site := Site{Name: "Test"}
	_, err := client.UpdateSite("site-123", site)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestClient_DeleteSite_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/sites/site-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"deleted": true}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	err := client.DeleteSite("site-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_DeleteSite_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"code": "not_found", "message": "Site not found"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	err := client.DeleteSite("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestClient_GetSite_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.GetSite("site-123")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestClient_GetDevice_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.GetDevice("device-123")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestClient_CreateDevice_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{SiteID: new("site-123"), IPAddress: "192.168.1.1"}
	_, err := client.CreateDevice(device)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestClient_UpdateDevice_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{SiteID: new("site-123"), IPAddress: "192.168.1.1"}
	_, err := client.UpdateDevice("device-123", device)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestClient_ConnectionError(t *testing.T) {
	client := NewClient("test-token", "http://localhost:99999")

	_, err := client.GetDevice("device-123")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestClient_APIErrorWithoutJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`Internal Server Error`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.GetDevice("device-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if errors.Is(err, ErrNotFound) {
		t.Error("did not expect ErrNotFound for 500 error")
	}

	if got := err.Error(); got != "API error (500): Internal Server Error" {
		t.Errorf("unexpected error rendering: %q", got)
	}
}

func TestClient_GetOrganization_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/organization" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"id": "org-123",
				"name": "Test ISP",
				"slug": "test-isp",
				"subscription_plan": "paid",
				"use_sites": true,
				"snmp_community_set": true,
				"snmp_version": "2c",
				"snmp_port": 161,
				"snmp_transport": "udp",
				"mikrotik_enabled": true,
				"mikrotik_port": 8729,
				"inserted_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-02-01T00:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	org, err := client.GetOrganization()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if org.SubscriptionPlan != "paid" {
		t.Errorf("expected subscription_plan paid, got %s", org.SubscriptionPlan)
	}
	if !org.SNMPCommunitySet {
		t.Error("expected snmp_community_set true")
	}
	if org.SNMPPort == nil || *org.SNMPPort != 161 {
		t.Errorf("expected snmp_port 161, got %v", org.SNMPPort)
	}
	if org.MikrotikEnabled == nil || !*org.MikrotikEnabled {
		t.Errorf("expected mikrotik_enabled true, got %v", org.MikrotikEnabled)
	}
	if org.UpdatedAt != "2024-02-01T00:00:00Z" {
		t.Errorf("expected updated_at 2024-02-01T00:00:00Z, got %s", org.UpdatedAt)
	}
}

func TestClient_UpdateOrganization_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": {"code": "forbidden", "message": "Only organization owners can update organization settings"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	_, err := client.UpdateOrganization(Organization{UseSites: true})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := "API error (403 forbidden): Only organization owners can update organization settings"
	if got := err.Error(); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
