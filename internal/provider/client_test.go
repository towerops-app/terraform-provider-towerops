package provider

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_ErrNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
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
			"id": "device-123",
			"site_id": "site-456",
			"ip_address": "192.168.1.1",
			"name": "Test Device",
			"inserted_at": "2024-01-01T00:00:00Z"
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
	if device.SiteID != "site-456" {
		t.Errorf("expected SiteID site-456, got %s", device.SiteID)
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
			"id": "new-device-id",
			"site_id": "site-456",
			"ip_address": "192.168.1.100",
			"name": "New Device",
			"monitoring_enabled": true,
			"snmp_enabled": true,
			"inserted_at": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{
		SiteID:    "site-456",
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
			"id": "device-123",
			"site_id": "site-456",
			"ip_address": "192.168.1.200",
			"name": "Updated Device",
			"inserted_at": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{
		SiteID:    "site-456",
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
		w.Write([]byte(`{"error": "device not found"}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{
		SiteID:    "site-456",
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
		w.Write([]byte(`{"error": "invalid request"}`))
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
}

func TestClient_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors": {"ip_address": "is invalid"}}`))
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)

	device := Device{
		SiteID:    "site-456",
		IPAddress: "invalid",
	}

	_, err := client.CreateDevice(device)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if errors.Is(err, ErrNotFound) {
		t.Error("did not expect ErrNotFound for validation error")
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
			"id": "site-123",
			"name": "Test Site",
			"location": "New York",
			"inserted_at": "2024-01-01T00:00:00Z"
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
		w.Write([]byte(`{"error": "site not found"}`))
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
			"id": "new-site-id",
			"name": "New Site",
			"location": "Boston",
			"inserted_at": "2024-01-01T00:00:00Z"
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
		w.Write([]byte(`{"error": "name is required"}`))
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
			"id": "site-123",
			"name": "Updated Site",
			"location": "Chicago",
			"inserted_at": "2024-01-01T00:00:00Z"
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
		w.Write([]byte(`{"error": "site not found"}`))
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

		w.WriteHeader(http.StatusNoContent)
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
		w.Write([]byte(`{"error": "site not found"}`))
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

	device := Device{SiteID: "site-123", IPAddress: "192.168.1.1"}
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

	device := Device{SiteID: "site-123", IPAddress: "192.168.1.1"}
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
}
