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

func TestAccOrganizationResource_basic(t *testing.T) {
	var mu sync.Mutex
	currentUseSites := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organization":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": Organization{
					ID:       "org-123",
					Name:     "Test ISP",
					Slug:     "test-isp",
					UseSites: currentUseSites,
				},
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/organization":
			var body map[string]Organization
			json.NewDecoder(r.Body).Decode(&body)
			currentUseSites = body["organization"].UseSites
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": Organization{
					ID:       "org-123",
					Name:     "Test ISP",
					Slug:     "test-isp",
					UseSites: currentUseSites,
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
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
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": Organization{
					ID:       "org-123",
					Name:     "Test ISP",
					Slug:     "test-isp",
					UseSites: currentUseSites,
				},
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/organization":
			var body map[string]Organization
			json.NewDecoder(r.Body).Decode(&body)
			currentUseSites = body["organization"].UseSites
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": Organization{
					ID:       "org-123",
					Name:     "Test ISP",
					Slug:     "test-isp",
					UseSites: currentUseSites,
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
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
