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

func TestAccSiteResource_basic(t *testing.T) {
	var siteID string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
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
				Config: testAccSiteResourceConfig(server.URL, "Test Site"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "name", "Test Site"),
					resource.TestCheckResourceAttrSet("towerops_site.test", "id"),
					resource.TestCheckResourceAttrSet("towerops_site.test", "inserted_at"),
				),
			},
		},
	})
}

func TestAccSiteResource_withLocation(t *testing.T) {
	var siteID string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		location := "New York, NY"

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Site With Location",
				Location:   &location,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Site With Location",
				Location:   &location,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
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
				Config: testAccSiteResourceConfigWithLocation(server.URL, "Site With Location", "New York, NY"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "name", "Site With Location"),
					resource.TestCheckResourceAttr("towerops_site.test", "location", "New York, NY"),
				),
			},
		},
	})
}

func TestAccSiteResource_update(t *testing.T) {
	var siteID string
	var currentName string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			currentName = "Original Name"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       currentName,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       currentName,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/sites/"+siteID:
			var body map[string]Site
			json.NewDecoder(r.Body).Decode(&body)
			currentName = body["site"].Name
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       currentName,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
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
				Config: testAccSiteResourceConfig(server.URL, "Original Name"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "name", "Original Name"),
				),
			},
			{
				Config: testAccSiteResourceConfig(server.URL, "Updated Name"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "name", "Updated Name"),
				),
			},
		},
	})
}

func TestAccSiteResource_importState(t *testing.T) {
	var siteID string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "imported-site-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Imported Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/imported-site-id":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Site{
				ID:         "imported-site-id",
				Name:       "Imported Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/imported-site-id":
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
				Config: testAccSiteResourceConfig(server.URL, "Imported Site"),
			},
			{
				ResourceName:      "towerops_site.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSiteResourceConfig(apiURL, name string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_site" "test" {
  name = %q
}
`, apiURL, name)
}

func testAccSiteResourceConfigWithLocation(apiURL, name, location string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_site" "test" {
  name     = %q
  location = %q
}
`, apiURL, name, location)
}

func TestAccSiteResource_createError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites" {
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
				Config:      testAccSiteResourceConfig(server.URL, ""),
				ExpectError: regexp.MustCompile(`Failed to create site`),
			},
		},
	})
}

func TestAccSiteResource_readError(t *testing.T) {
	var siteID string
	var mu sync.Mutex
	readCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			readCount++
			if readCount > 1 {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
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
				Config:      testAccSiteResourceConfig(server.URL, "Test Site"),
				ExpectError: regexp.MustCompile(`Failed to read site`),
			},
		},
	})
}

func TestAccSiteResource_updateError(t *testing.T) {
	var siteID string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Original Name",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Original Name",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/sites/"+siteID:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "update failed"}`))

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
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
				Config: testAccSiteResourceConfig(server.URL, "Original Name"),
			},
			{
				Config:      testAccSiteResourceConfig(server.URL, "Updated Name"),
				ExpectError: regexp.MustCompile(`Failed to update site`),
			},
		},
	})
}

func TestAccSiteResource_deleteError(t *testing.T) {
	var siteID string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "delete failed"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config:  testAccSiteResourceConfig(server.URL, "Test Site"),
				Destroy: true,
			},
		},
		ErrorCheck: func(err error) error {
			if err != nil && regexp.MustCompile(`Failed to delete site`).MatchString(err.Error()) {
				return nil
			}
			return err
		},
	})
}

func TestAccSiteResource_withSNMPCommunity(t *testing.T) {
	var siteID string
	var mu sync.Mutex
	community := "public"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Site{
				ID:            siteID,
				Name:          "SNMP Site",
				SNMPCommunity: &community,
				InsertedAt:    "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Site{
				ID:            siteID,
				Name:          "SNMP Site",
				SNMPCommunity: &community,
				InsertedAt:    "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
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
				Config: testAccSiteResourceConfigWithSNMPCommunity(server.URL, "SNMP Site", "public"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "name", "SNMP Site"),
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community", "public"),
				),
			},
		},
	})
}

func testAccSiteResourceConfigWithSNMPCommunity(apiURL, name, community string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_site" "test" {
  name           = %q
  snmp_community = %q
}
`, apiURL, name, community)
}
