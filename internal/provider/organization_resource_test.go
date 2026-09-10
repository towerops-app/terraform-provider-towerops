package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// writeOrgData answers with the success envelope every /api/v1 endpoint uses.
func writeOrgData(w http.ResponseWriter, status int, org Organization) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{"data": org})
}

// writeOrgError answers with the failure envelope every /api/v1 endpoint uses.
func writeOrgError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	})
}

// decodeOrgRequest reads the {"organization": {...}} request wrapper.
func decodeOrgRequest(r *http.Request) Organization {
	var body map[string]Organization
	json.NewDecoder(r.Body).Decode(&body)
	return body["organization"]
}

func TestAccOrganizationResource_basic(t *testing.T) {
	var mu sync.Mutex
	currentUseSites := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organization":
			writeOrgData(w, http.StatusOK, Organization{
				ID:       "org-123",
				Name:     "Test ISP",
				Slug:     "test-isp",
				UseSites: currentUseSites,
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/organization":
			currentUseSites = decodeOrgRequest(r).UseSites
			writeOrgData(w, http.StatusOK, Organization{
				ID:       "org-123",
				Name:     "Test ISP",
				Slug:     "test-isp",
				UseSites: currentUseSites,
			})

		default:
			writeOrgError(w, http.StatusNotFound, "not_found", "Resource not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationResourceConfig(server.URL, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_organization.settings", "use_sites", "true"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "name", "Test ISP"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "slug", "test-isp"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmp_community_set", "false"),
				),
			},
		},
	})
}

func TestAccOrganizationResource_update(t *testing.T) {
	var mu sync.Mutex
	currentUseSites := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organization":
			writeOrgData(w, http.StatusOK, Organization{
				ID:       "org-123",
				Name:     "Test ISP",
				Slug:     "test-isp",
				UseSites: currentUseSites,
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/organization":
			currentUseSites = decodeOrgRequest(r).UseSites
			writeOrgData(w, http.StatusOK, Organization{
				ID:       "org-123",
				Name:     "Test ISP",
				Slug:     "test-isp",
				UseSites: currentUseSites,
			})

		default:
			writeOrgError(w, http.StatusNotFound, "not_found", "Resource not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationResourceConfig(server.URL, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_organization.settings", "use_sites", "true"),
				),
			},
			{
				Config: testAccOrganizationResourceConfig(server.URL, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_organization.settings", "use_sites", "false"),
				),
			},
		},
	})
}

// TestAccOrganizationResource_writeOnlySecrets pins the behaviour the API
// forces on us: snmp_community and the SNMPv3 password go up on write and
// never come back down. The provider has to keep the configured value in
// state instead of overwriting it with the empty response field, and it has
// to surface the companion booleans so drift is still visible.
func TestAccOrganizationResource_writeOnlySecrets(t *testing.T) {
	var mu sync.Mutex
	stored := Organization{
		ID:            "org-123",
		Name:          "Test ISP",
		Slug:          "test-isp",
		UseSites:      true,
		SNMPTransport: new("udp"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organization":
			writeOrgData(w, http.StatusOK, stored)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/organization":
			sent := decodeOrgRequest(r)
			if sent.SnmpCommunity == "" {
				writeOrgError(w, http.StatusUnprocessableEntity, "validation_error", "Validation failed")
				return
			}

			// Mirror the API: record that the secrets are stored, apply the
			// settings that were sent, keep the server-side defaults for the
			// ones that were not, and never return the secrets themselves.
			stored.UseSites = sent.UseSites
			stored.SNMPCommunitySet = true
			if sent.SNMPVersion != nil {
				stored.SNMPVersion = sent.SNMPVersion
			}
			if sent.SNMPPort != nil {
				stored.SNMPPort = sent.SNMPPort
			}
			if sent.SNMPTransport != nil {
				stored.SNMPTransport = sent.SNMPTransport
			}
			if sent.SNMPv3Username != nil {
				stored.SNMPv3Username = sent.SNMPv3Username
			}
			writeOrgData(w, http.StatusOK, stored)

		default:
			writeOrgError(w, http.StatusNotFound, "not_found", "Resource not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationResourceSecretsConfig(server.URL, 1161),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmp_community", "super-secret"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmpv3_auth_password", "hunter2hunter2"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmp_community_set", "true"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmp_version", "3"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmp_port", "1161"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmpv3_username", "monitor"),
					// Never configured, so the API value has to fill it in.
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmp_transport", "udp"),
				),
			},
			{
				// The update path has to keep the secrets too: it sends them
				// and gets a response that cannot echo them back.
				Config: testAccOrganizationResourceSecretsConfig(server.URL, 2161),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmp_port", "2161"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmp_community", "super-secret"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmpv3_auth_password", "hunter2hunter2"),
					resource.TestCheckResourceAttr("towerops_organization.settings", "snmp_community_set", "true"),
				),
			},
		},
	})
}

func testAccOrganizationResourceConfig(apiURL string, useSites bool) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_organization" "settings" {
  use_sites = %t
}
`, apiURL, useSites)
}

func testAccOrganizationResourceSecretsConfig(apiURL string, snmpPort int) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_organization" "settings" {
  use_sites            = true
  snmp_community       = "super-secret"
  snmp_version         = "3"
  snmp_port            = %d
  snmpv3_username      = "monitor"
  snmpv3_auth_password = "hunter2hunter2"
}
`, apiURL, snmpPort)
}
