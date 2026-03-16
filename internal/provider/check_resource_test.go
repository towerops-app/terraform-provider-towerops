package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCheckResource_http(t *testing.T) {
	var checkID string
	var mu sync.Mutex
	enabled := true
	interval := 60
	timeout := 5000
	retry := 30
	maxAttempts := 3
	currentState := 3
	stateType := "soft"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = "test-http-check-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 "Web Health",
				CheckType:            "http",
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"url":              "https://example.com/health",
					"method":           "GET",
					"expected_status":  200,
					"verify_ssl":       true,
					"follow_redirects": true,
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 "Web Health",
				CheckType:            "http",
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"url":              "https://example.com/health",
					"method":           "GET",
					"expected_status":  200,
					"verify_ssl":       true,
					"follow_redirects": true,
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceHTTP(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_check.test", "name", "Web Health"),
					resource.TestCheckResourceAttr("towerops_check.test", "check_type", "http"),
					resource.TestCheckResourceAttr("towerops_check.test", "url", "https://example.com/health"),
					resource.TestCheckResourceAttr("towerops_check.test", "expected_status", "200"),
					resource.TestCheckResourceAttr("towerops_check.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("towerops_check.test", "id"),
					resource.TestCheckResourceAttrSet("towerops_check.test", "inserted_at"),
				),
			},
		},
	})
}

func TestAccCheckResource_tcp(t *testing.T) {
	var checkID string
	var mu sync.Mutex
	enabled := true
	interval := 60
	timeout := 5000
	retry := 30
	maxAttempts := 3
	currentState := 3
	stateType := "soft"
	deviceID := "device-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = "test-tcp-check-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 "MySQL Port",
				CheckType:            "tcp",
				DeviceID:             &deviceID,
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"host": "10.0.0.5",
					"port": 3306,
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 "MySQL Port",
				CheckType:            "tcp",
				DeviceID:             &deviceID,
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"host": "10.0.0.5",
					"port": 3306,
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceTCP(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_check.test", "name", "MySQL Port"),
					resource.TestCheckResourceAttr("towerops_check.test", "check_type", "tcp"),
					resource.TestCheckResourceAttr("towerops_check.test", "host", "10.0.0.5"),
					resource.TestCheckResourceAttr("towerops_check.test", "port", "3306"),
					resource.TestCheckResourceAttr("towerops_check.test", "device_id", "device-123"),
				),
			},
		},
	})
}

func TestAccCheckResource_dns(t *testing.T) {
	var checkID string
	var mu sync.Mutex
	enabled := true
	interval := 60
	timeout := 5000
	retry := 30
	maxAttempts := 3
	currentState := 3
	stateType := "soft"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = "test-dns-check-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 "DNS Resolution",
				CheckType:            "dns",
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"hostname":    "google.com",
					"server":      "8.8.8.8",
					"record_type": "A",
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 "DNS Resolution",
				CheckType:            "dns",
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"hostname":    "google.com",
					"server":      "8.8.8.8",
					"record_type": "A",
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceDNS(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_check.test", "name", "DNS Resolution"),
					resource.TestCheckResourceAttr("towerops_check.test", "check_type", "dns"),
					resource.TestCheckResourceAttr("towerops_check.test", "hostname", "google.com"),
					resource.TestCheckResourceAttr("towerops_check.test", "dns_server", "8.8.8.8"),
					resource.TestCheckResourceAttr("towerops_check.test", "record_type", "A"),
				),
			},
		},
	})
}

func TestAccCheckResource_ping(t *testing.T) {
	var checkID string
	var mu sync.Mutex
	enabled := true
	interval := 60
	timeout := 5000
	retry := 30
	maxAttempts := 3
	currentState := 3
	stateType := "soft"
	deviceID := "device-gw"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = "test-ping-check-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 "Gateway Ping",
				CheckType:            "ping",
				DeviceID:             &deviceID,
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"host":  "10.0.0.1",
					"count": 3,
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 "Gateway Ping",
				CheckType:            "ping",
				DeviceID:             &deviceID,
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"host":  "10.0.0.1",
					"count": 3,
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourcePing(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_check.test", "name", "Gateway Ping"),
					resource.TestCheckResourceAttr("towerops_check.test", "check_type", "ping"),
					resource.TestCheckResourceAttr("towerops_check.test", "host", "10.0.0.1"),
					resource.TestCheckResourceAttr("towerops_check.test", "ping_count", "3"),
					resource.TestCheckResourceAttr("towerops_check.test", "device_id", "device-gw"),
				),
			},
		},
	})
}

func TestAccCheckResource_update(t *testing.T) {
	var checkID string
	var currentName string
	var mu sync.Mutex
	enabled := true
	interval := 60
	timeout := 5000
	retry := 30
	maxAttempts := 3
	currentState := 3
	stateType := "soft"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = "test-update-check-id"
			currentName = "Original"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 currentName,
				CheckType:            "http",
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"url":              "https://example.com",
					"method":           "GET",
					"expected_status":  200,
					"verify_ssl":       true,
					"follow_redirects": true,
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 currentName,
				CheckType:            "http",
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"url":              "https://example.com",
					"method":           "GET",
					"expected_status":  200,
					"verify_ssl":       true,
					"follow_redirects": true,
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/checks/"+checkID:
			var body map[string]Check
			json.NewDecoder(r.Body).Decode(&body)
			currentName = body["check"].Name
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Check{
				ID:                   checkID,
				Name:                 currentName,
				CheckType:            "http",
				Enabled:              &enabled,
				IntervalSeconds:      &interval,
				TimeoutMs:            &timeout,
				RetryIntervalSeconds: &retry,
				MaxCheckAttempts:     &maxAttempts,
				CurrentState:         &currentState,
				CurrentStateType:     &stateType,
				Config: map[string]any{
					"url":              "https://example.com",
					"method":           "GET",
					"expected_status":  200,
					"verify_ssl":       true,
					"follow_redirects": true,
				},
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceHTTPNamed(server.URL, "Original"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_check.test", "name", "Original"),
				),
			},
			{
				Config: testAccCheckResourceHTTPNamed(server.URL, "Updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_check.test", "name", "Updated"),
				),
			},
		},
	})
}

func TestAccCheckResource_recreateOn404(t *testing.T) {
	var checkID string
	var checkDeleted bool
	var mu sync.Mutex
	enabled := true
	interval := 60
	timeout := 5000
	retry := 30
	maxAttempts := 3
	currentState := 3
	stateType := "soft"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		checkResp := Check{
			ID:                   "new-check-id",
			Name:                 "Web Health",
			CheckType:            "http",
			Enabled:              &enabled,
			IntervalSeconds:      &interval,
			TimeoutMs:            &timeout,
			RetryIntervalSeconds: &retry,
			MaxCheckAttempts:     &maxAttempts,
			CurrentState:         &currentState,
			CurrentStateType:     &stateType,
			Config: map[string]any{
				"url":              "https://example.com/health",
				"method":           "GET",
				"expected_status":  200,
				"verify_ssl":       true,
				"follow_redirects": true,
			},
			InsertedAt: "2024-01-01T00:00:00Z",
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = "new-check-id"
			checkDeleted = false
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(checkResp)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			if checkDeleted {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "check not found"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(checkResp)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/checks/"+checkID:
			if checkDeleted {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "check not found"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(checkResp)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			checkDeleted = true
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceHTTP(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_check.test", "name", "Web Health"),
				),
			},
			{
				PreConfig: func() {
					mu.Lock()
					checkDeleted = true
					mu.Unlock()
				},
				Config: testAccCheckResourceHTTP(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("towerops_check.test", "id"),
				),
			},
		},
	})
}

func TestAccCheckResource_importState(t *testing.T) {
	var checkID string
	var mu sync.Mutex
	enabled := true
	interval := 60
	timeout := 5000
	retry := 30
	maxAttempts := 3
	currentState := 3
	stateType := "soft"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		checkResp := Check{
			ID:                   "imported-check-id",
			Name:                 "Web Health",
			CheckType:            "http",
			Enabled:              &enabled,
			IntervalSeconds:      &interval,
			TimeoutMs:            &timeout,
			RetryIntervalSeconds: &retry,
			MaxCheckAttempts:     &maxAttempts,
			CurrentState:         &currentState,
			CurrentStateType:     &stateType,
			Config: map[string]any{
				"url":              "https://example.com/health",
				"method":           "GET",
				"expected_status":  200,
				"verify_ssl":       true,
				"follow_redirects": true,
			},
			InsertedAt: "2024-01-01T00:00:00Z",
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = "imported-check-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(checkResp)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/imported-check-id":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(checkResp)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/imported-check-id":
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	_ = checkID

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceHTTP(server.URL),
			},
			{
				ResourceName:            "towerops_check.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ping_count", "record_type"},
			},
		},
	})
}

func TestAccCheckResource_createError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "name is required"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckResourceHTTP(server.URL),
				ExpectError: regexp.MustCompile(`Failed to create check`),
			},
		},
	})
}

// HCL config helpers

func testAccCheckResourceHTTP(apiURL string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_check" "test" {
  name            = "Web Health"
  check_type      = "http"
  url             = "https://example.com/health"
  expected_status = 200
}
`, apiURL)
}

func testAccCheckResourceHTTPNamed(apiURL, name string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_check" "test" {
  name            = %q
  check_type      = "http"
  url             = "https://example.com"
  expected_status = 200
}
`, apiURL, name)
}

func testAccCheckResourceTCP(apiURL string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_check" "test" {
  name       = "MySQL Port"
  check_type = "tcp"
  device_id  = "device-123"
  host       = "10.0.0.5"
  port       = 3306
}
`, apiURL)
}

func testAccCheckResourceDNS(apiURL string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_check" "test" {
  name        = "DNS Resolution"
  check_type  = "dns"
  hostname    = "google.com"
  dns_server  = "8.8.8.8"
  record_type = "A"
}
`, apiURL)
}

func testAccCheckResourcePing(apiURL string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_check" "test" {
  name       = "Gateway Ping"
  check_type = "ping"
  device_id  = "device-gw"
  host       = "10.0.0.1"
  ping_count = 3
}
`, apiURL)
}
