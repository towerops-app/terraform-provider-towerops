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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// writeWebhookEndpointData emits the `{"data": ...}` success envelope every
// /api/v1 endpoint returns.
func writeWebhookEndpointData(w http.ResponseWriter, status int, endpoint WebhookEndpoint) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]WebhookEndpoint{"data": endpoint})
}

// writeWebhookEndpointError emits the `{"error": {...}}` failure envelope.
func writeWebhookEndpointError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}

func writeWebhookEndpointDeleted(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"data":{"deleted":true}}`))
}

func decodeWebhookEndpointRequest(t *testing.T, r *http.Request) WebhookEndpoint {
	t.Helper()

	var body map[string]WebhookEndpoint
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("failed to decode request body: %v", err)
	}
	return body["webhook_endpoint"]
}

// TestAccWebhookEndpointResource_configuredSecretSurvivesRead proves the
// write-only rule: a show response omits the secret and must not clear the
// configured value from state.
func TestAccWebhookEndpointResource_configuredSecretSurvivesRead(t *testing.T) {
	const endpointID = "test-webhook-endpoint-id"
	const configuredSecret = "s3cret-value-abcdef"

	var mu sync.Mutex
	var sentSecret string
	failures := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/webhook_endpoints":
			sent := decodeWebhookEndpointRequest(t, r)
			if sent.Secret != nil {
				sentSecret = *sent.Secret
			}
			// The create response is the only one that carries the secret.
			writeWebhookEndpointData(w, http.StatusCreated, WebhookEndpoint{
				ID:                  endpointID,
				URL:                 sent.URL,
				Secret:              sent.Secret,
				Events:              sent.Events,
				Active:              new(true),
				ConsecutiveFailures: &failures,
				CreatedAt:           "2026-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/webhook_endpoints/"+endpointID:
			// Show never returns the secret.
			writeWebhookEndpointData(w, http.StatusOK, WebhookEndpoint{
				ID:                  endpointID,
				URL:                 "https://example.com/hooks/towerops",
				Events:              []string{"alert.triggered"},
				Active:              new(true),
				ConsecutiveFailures: &failures,
				CreatedAt:           "2026-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/webhook_endpoints/"+endpointID:
			writeWebhookEndpointDeleted(w)

		default:
			writeWebhookEndpointError(w, http.StatusNotFound, "not_found", "Webhook endpoint not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookEndpointResourceConfigWithSecret(server.URL, configuredSecret),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "id", endpointID),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "url", "https://example.com/hooks/towerops"),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "events.#", "1"),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "events.0", "alert.triggered"),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "secret", configuredSecret),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "active", "true"),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "consecutive_failures", "0"),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "created_at", "2026-01-01T00:00:00Z"),
					func(*terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if sentSecret != configuredSecret {
							return fmt.Errorf("create request carried secret %q, want %q", sentSecret, configuredSecret)
						}
						return nil
					},
				),
			},
			{
				// Refresh against a payload without a secret: the configured
				// value must still be in state and the plan must stay empty.
				RefreshState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "secret", configuredSecret),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "url", "https://example.com/hooks/towerops"),
				),
			},
		},
	})
}

// TestAccWebhookEndpointResource_generatedSecret proves the other half: with no
// secret in configuration, the server generated one is captured from the create
// response and kept across a read that omits it.
func TestAccWebhookEndpointResource_generatedSecret(t *testing.T) {
	const endpointID = "generated-secret-endpoint-id"
	const generatedSecret = "server-generated-secret-0001"

	var mu sync.Mutex
	failures := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/webhook_endpoints":
			sent := decodeWebhookEndpointRequest(t, r)
			if sent.Secret != nil {
				t.Errorf("create request carried a secret %q, want none", *sent.Secret)
			}

			writeWebhookEndpointData(w, http.StatusCreated, WebhookEndpoint{
				ID:                  endpointID,
				URL:                 sent.URL,
				Secret:              new(generatedSecret),
				Events:              sent.Events,
				Active:              new(true),
				ConsecutiveFailures: &failures,
				CreatedAt:           "2026-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/webhook_endpoints/"+endpointID:
			writeWebhookEndpointData(w, http.StatusOK, WebhookEndpoint{
				ID:                  endpointID,
				URL:                 "https://example.com/hooks/towerops",
				Events:              []string{"alert.triggered"},
				Active:              new(true),
				ConsecutiveFailures: &failures,
				CreatedAt:           "2026-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/webhook_endpoints/"+endpointID:
			writeWebhookEndpointDeleted(w)

		default:
			writeWebhookEndpointError(w, http.StatusNotFound, "not_found", "Webhook endpoint not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookEndpointResourceConfig(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "secret", generatedSecret),
				),
			},
			{
				RefreshState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "secret", generatedSecret),
				),
			},
		},
	})
}

func TestAccWebhookEndpointResource_updateEvents(t *testing.T) {
	const endpointID = "update-events-endpoint-id"

	var mu sync.Mutex
	var currentEvents []string
	failures := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		respond := func(status int, secret *string) {
			writeWebhookEndpointData(w, status, WebhookEndpoint{
				ID:                  endpointID,
				URL:                 "https://example.com/hooks/towerops",
				Secret:              secret,
				Events:              currentEvents,
				Active:              new(true),
				ConsecutiveFailures: &failures,
				CreatedAt:           "2026-01-01T00:00:00Z",
			})
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/webhook_endpoints":
			sent := decodeWebhookEndpointRequest(t, r)
			currentEvents = sent.Events
			respond(http.StatusCreated, sent.Secret)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/webhook_endpoints/"+endpointID:
			respond(http.StatusOK, nil)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/webhook_endpoints/"+endpointID:
			sent := decodeWebhookEndpointRequest(t, r)
			currentEvents = sent.Events
			respond(http.StatusOK, nil)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/webhook_endpoints/"+endpointID:
			writeWebhookEndpointDeleted(w)

		default:
			writeWebhookEndpointError(w, http.StatusNotFound, "not_found", "Webhook endpoint not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookEndpointResourceConfigWithEvents(server.URL, `["alert.triggered"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "events.#", "1"),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "events.0", "alert.triggered"),
				),
			},
			{
				Config: testAccWebhookEndpointResourceConfigWithEvents(server.URL, `["alert.triggered", "alert.resolved", "maintenance.started"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "events.#", "3"),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "events.0", "alert.triggered"),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "events.1", "alert.resolved"),
					resource.TestCheckResourceAttr("towerops_webhook_endpoint.test", "events.2", "maintenance.started"),
					func(*terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if len(currentEvents) != 3 {
							return fmt.Errorf("server holds %d events, want 3", len(currentEvents))
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccWebhookEndpointResource_insufficientScope covers a token without the
// webhooks:manage scope.
func TestAccWebhookEndpointResource_insufficientScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeWebhookEndpointError(
			w,
			http.StatusForbidden,
			"insufficient_scope",
			"Token is missing the required scope: webhooks:manage",
		)
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config:      testAccWebhookEndpointResourceConfig(server.URL),
				ExpectError: regexp.MustCompile(`insufficient_scope`),
			},
		},
	})
}

func testAccWebhookEndpointResourceConfig(apiURL string) string {
	return testAccWebhookEndpointResourceConfigWithEvents(apiURL, `["alert.triggered"]`)
}

func testAccWebhookEndpointResourceConfigWithEvents(apiURL, events string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_webhook_endpoint" "test" {
  url    = "https://example.com/hooks/towerops"
  events = %s
}
`, apiURL, events)
}

func testAccWebhookEndpointResourceConfigWithSecret(apiURL, secret string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_webhook_endpoint" "test" {
  url    = "https://example.com/hooks/towerops"
  events = ["alert.triggered"]
  secret = %q
}
`, apiURL, secret)
}
