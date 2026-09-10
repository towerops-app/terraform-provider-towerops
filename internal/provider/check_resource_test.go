package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// checkTestData emits the `{"data": ...}` success envelope every /api/v1
// endpoint returns.
func checkTestData(w http.ResponseWriter, status int, check Check) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]Check{"data": check})
}

// checkTestDeleted emits the delete envelope the shared resource helper returns.
func checkTestDeleted(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"data":{"deleted":true}}`)
}

// checkTestError emits the API failure envelope.
func checkTestError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"code":%q,"message":%q}}`, code, message)
}

func TestAccCheckResource_http(t *testing.T) {
	var checkID string
	var mu sync.Mutex
	enabled := true
	alerting := true
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
			ID:                   "test-http-check-id",
			Name:                 "Web Health",
			CheckType:            "http",
			Enabled:              &enabled,
			Alerting:             &alerting,
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
			checkID = checkResp.ID
			checkTestData(w, http.StatusCreated, checkResp)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestData(w, http.StatusOK, checkResp)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestDeleted(w)

		default:
			checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
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
					resource.TestCheckResourceAttr("towerops_check.test", "alerting", "true"),
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
	alerting := true
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

		checkResp := Check{
			ID:                   "test-tcp-check-id",
			Name:                 "MySQL Port",
			CheckType:            "tcp",
			DeviceID:             &deviceID,
			Enabled:              &enabled,
			Alerting:             &alerting,
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
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = checkResp.ID
			checkTestData(w, http.StatusCreated, checkResp)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestData(w, http.StatusOK, checkResp)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestDeleted(w)

		default:
			checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
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
	var sentServer string
	enabled := true
	alerting := true
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
			ID:                   "test-dns-check-id",
			Name:                 "DNS Resolution",
			CheckType:            "dns",
			Enabled:              &enabled,
			Alerting:             &alerting,
			IntervalSeconds:      &interval,
			TimeoutMs:            &timeout,
			RetryIntervalSeconds: &retry,
			MaxCheckAttempts:     &maxAttempts,
			CurrentState:         &currentState,
			CurrentStateType:     &stateType,
			Config: map[string]any{
				"hostname":    "google.com",
				"server":      sentServer,
				"record_type": "A",
			},
			InsertedAt: "2024-01-01T00:00:00Z",
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			// The DNS executor reads the nameserver from config["server"], so
			// that is the key the provider has to write dns_server into.
			var body map[string]Check
			json.NewDecoder(r.Body).Decode(&body)
			sentServer, _ = body["check"].Config["server"].(string)
			checkResp.Config["server"] = sentServer
			checkID = checkResp.ID
			checkTestData(w, http.StatusCreated, checkResp)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestData(w, http.StatusOK, checkResp)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestDeleted(w)

		default:
			checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
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
	alerting := true
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

		checkResp := Check{
			ID:                   "test-ping-check-id",
			Name:                 "Gateway Ping",
			CheckType:            "ping",
			DeviceID:             &deviceID,
			Enabled:              &enabled,
			Alerting:             &alerting,
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
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = checkResp.ID
			checkTestData(w, http.StatusCreated, checkResp)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestData(w, http.StatusOK, checkResp)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestDeleted(w)

		default:
			checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
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
	alerting := true
	interval := 60
	timeout := 5000
	retry := 30
	maxAttempts := 3
	currentState := 3
	stateType := "soft"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		respond := func(status int) {
			checkTestData(w, status, Check{
				ID:                   checkID,
				Name:                 currentName,
				CheckType:            "http",
				Enabled:              &enabled,
				Alerting:             &alerting,
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
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			checkID = "test-update-check-id"
			currentName = "Original"
			respond(http.StatusCreated)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			respond(http.StatusOK)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/checks/"+checkID:
			var body map[string]Check
			json.NewDecoder(r.Body).Decode(&body)
			currentName = body["check"].Name
			respond(http.StatusOK)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestDeleted(w)

		default:
			checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
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

// TestAccCheckResource_alerting pins the arming contract: the create endpoint
// ignores `alerting` (a new check is always armed), and only the update endpoint
// honors it.
func TestAccCheckResource_alerting(t *testing.T) {
	var checkID string
	var mu sync.Mutex
	var createSentAlerting bool
	enabled := true
	armed := true
	interval := 60
	timeout := 5000
	retry := 30
	maxAttempts := 3
	currentState := 3
	stateType := "soft"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		respond := func(status int) {
			checkTestData(w, status, Check{
				ID:                   checkID,
				Name:                 "Web Health",
				CheckType:            "http",
				Enabled:              &enabled,
				Alerting:             &armed,
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
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			var body map[string]Check
			json.NewDecoder(r.Body).Decode(&body)
			createSentAlerting = body["check"].Alerting != nil
			checkID = "test-alerting-check-id"
			armed = true
			respond(http.StatusCreated)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			respond(http.StatusOK)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/checks/"+checkID:
			var body map[string]Check
			json.NewDecoder(r.Body).Decode(&body)
			if body["check"].Alerting != nil {
				armed = *body["check"].Alerting
			}
			respond(http.StatusOK)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			checkTestDeleted(w)

		default:
			checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceHTTP(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_check.test", "alerting", "true"),
					func(*terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if createSentAlerting {
							return fmt.Errorf("create request carried alerting, the API ignores it there")
						}
						return nil
					},
				),
			},
			{
				Config: testAccCheckResourceHTTPAlerting(server.URL, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_check.test", "alerting", "false"),
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
	alerting := true
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
			Alerting:             &alerting,
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
			checkTestData(w, http.StatusCreated, checkResp)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+checkID:
			if checkDeleted {
				checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
				return
			}
			checkTestData(w, http.StatusOK, checkResp)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/checks/"+checkID:
			if checkDeleted {
				checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
				return
			}
			checkTestData(w, http.StatusOK, checkResp)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+checkID:
			checkDeleted = true
			checkTestDeleted(w)

		default:
			checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
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
	alerting := true
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
			Alerting:             &alerting,
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
			checkTestData(w, http.StatusCreated, checkResp)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/imported-check-id":
			checkTestData(w, http.StatusOK, checkResp)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/imported-check-id":
			checkTestDeleted(w)

		default:
			checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprint(w, `{"error":{"code":"validation_error","message":"Validation failed","details":{"fields":{"name":["can't be blank"]}}}}`)
			return
		}
		checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckResourceHTTP(server.URL),
				ExpectError: regexp.MustCompile(`(?s)Failed to create check.*validation_error`),
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

func testAccCheckResourceHTTPAlerting(apiURL string, alerting bool) string {
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
  alerting        = %t
}
`, apiURL, alerting)
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

// checkTestEchoServer serves back whatever config the provider sent, and hands
// the caller a reader for that config.
//
// Asserting on state alone would prove nothing here: an Optional attribute
// keeps its planned value in state whether or not the resource ever put it on
// the wire, so a dropped `config` mapping still produces a green plan. The
// observable contract is that configuring the attribute makes the value reach
// the API, so that is what these tests read.
func checkTestEchoServer(t *testing.T, id, checkType string) (*httptest.Server, func() map[string]any) {
	t.Helper()

	var mu sync.Mutex
	var stored map[string]any
	enabled := true
	alerting := true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		respond := func(status int) {
			checkTestData(w, status, Check{
				ID:         id,
				Name:       "Echo",
				CheckType:  checkType,
				Enabled:    &enabled,
				Alerting:   &alerting,
				Config:     stored,
				InsertedAt: "2024-01-01T00:00:00Z",
			})
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/checks":
			var body map[string]Check
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create body: %v", err)
			}
			stored = body["check"].Config
			respond(http.StatusCreated)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/checks/"+id:
			respond(http.StatusOK)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/checks/"+id:
			checkTestDeleted(w)

		default:
			checkTestError(w, http.StatusNotFound, "not_found", "Check not found")
		}
	}))

	return server, func() map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return stored
	}
}

// checkTestSentConfig asserts the value the provider put at config[key].
func checkTestSentConfig(t *testing.T, sent func() map[string]any, key string, want any) resource.TestCheckFunc {
	t.Helper()

	return func(*terraform.State) error {
		got, ok := sent()[key]
		if !ok {
			return fmt.Errorf("config[%q] was never sent to the API", key)
		}
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("config[%q] = %#v, want %#v", key, got, want)
		}
		return nil
	}
}

func TestAccCheckResource_httpHeadersAndBody(t *testing.T) {
	server, sent := checkTestEchoServer(t, "test-http-echo-id", "http")
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_check" "test" {
  name         = "Echo"
  check_type   = "http"
  url          = "https://example.com/ingest"
  method       = "POST"
  request_body = "{\"probe\":true}"

  request_headers = {
    "X-Probe"   = "towerops"
    "X-Api-Key" = "abc123"
  }
}
`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkTestSentConfig(t, sent, "body", `{"probe":true}`),
					checkTestSentConfig(t, sent, "headers", map[string]any{
						"X-Probe":   "towerops",
						"X-Api-Key": "abc123",
					}),
					resource.TestCheckResourceAttr("towerops_check.test", "request_headers.X-Probe", "towerops"),
				),
			},
		},
	})
}

func TestAccCheckResource_pingThresholds(t *testing.T) {
	server, sent := checkTestEchoServer(t, "test-ping-echo-id", "ping")
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_check" "test" {
  name       = "Echo"
  check_type = "ping"
  host       = "10.0.0.1"
  ping_count = 5

  loss_warning_percent = 2.5
  latency_warning_ms   = 100
  latency_critical_ms  = 250
}
`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkTestSentConfig(t, sent, "count", float64(5)),
					checkTestSentConfig(t, sent, "loss_warning_percent", 2.5),
					checkTestSentConfig(t, sent, "latency_warning_ms", float64(100)),
					checkTestSentConfig(t, sent, "latency_critical_ms", float64(250)),
				),
			},
		},
	})
}
