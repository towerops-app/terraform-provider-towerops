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

// writeSiteData renders the `{"data": <site>}` success envelope every
// /api/v1 endpoint answers with.
func writeSiteData(w http.ResponseWriter, status int, site Site) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]Site{"data": site})
}

// writeSiteDeleted renders the shared delete response, `{"data":{"deleted":true}}`.
func writeSiteDeleted(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"data":{"deleted":true}}`))
}

// writeSiteError renders the `{"error":{"code","message"}}` failure envelope.
func writeSiteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"error":{"code":%q,"message":%q}}`, code, message)
}

// decodeSiteRequest reads the `{"site": {...}}` request wrapper.
func decodeSiteRequest(r *http.Request) Site {
	var body map[string]Site
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body["site"]
}

func TestAccSiteResource_basic(t *testing.T) {
	var siteID string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			writeSiteData(w, http.StatusCreated, Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
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
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community_set", "false"),
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
			writeSiteData(w, http.StatusCreated, Site{
				ID:         siteID,
				Name:       "Site With Location",
				Location:   &location,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       "Site With Location",
				Location:   &location,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
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
			writeSiteData(w, http.StatusCreated, Site{
				ID:         siteID,
				Name:       currentName,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       currentName,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/sites/"+siteID:
			currentName = decodeSiteRequest(r).Name
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       currentName,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
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
			writeSiteData(w, http.StatusCreated, Site{
				ID:         siteID,
				Name:       "Imported Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/imported-site-id":
			writeSiteData(w, http.StatusOK, Site{
				ID:         "imported-site-id",
				Name:       "Imported Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/imported-site-id":
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
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

// TestAccSiteResource_allAttributes covers the full set of keys the API
// returns: the configured values must survive apply untouched, and the
// server-assigned agent_token_id must land in state.
func TestAccSiteResource_allAttributes(t *testing.T) {
	var siteID string
	var mu sync.Mutex

	agentTokenID := "11111111-1111-1111-1111-111111111111"
	parentSiteID := "22222222-2222-2222-2222-222222222222"

	respond := func(w http.ResponseWriter, status int, requested Site) {
		site := requested
		site.ID = siteID
		site.SNMPCommunity = nil
		site.SNMPCommunitySet = requested.SNMPCommunity != nil
		site.AgentTokenID = &agentTokenID
		site.InsertedAt = "2024-01-01T00:00:00Z"
		writeSiteData(w, status, site)
	}

	var stored Site

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "full-site-id"
			stored = decodeSiteRequest(r)
			respond(w, http.StatusCreated, stored)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			respond(w, http.StatusOK, stored)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
		}
	}))
	defer server.Close()

	config := fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_site" "test" {
  name           = "Full Site"
  description    = "Every attribute set"
  location       = "Verona, TX"
  address        = "123 Main St, Verona, TX 75482"
  latitude       = 33.4356
  longitude      = -96.0028
  display_order  = 7
  snmp_version   = "2c"
  snmp_port      = 1161
  snmp_transport = "udp"
  parent_site_id = %q
}
`, server.URL, parentSiteID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "name", "Full Site"),
					resource.TestCheckResourceAttr("towerops_site.test", "description", "Every attribute set"),
					resource.TestCheckResourceAttr("towerops_site.test", "location", "Verona, TX"),
					resource.TestCheckResourceAttr("towerops_site.test", "address", "123 Main St, Verona, TX 75482"),
					resource.TestCheckResourceAttr("towerops_site.test", "latitude", "33.4356"),
					resource.TestCheckResourceAttr("towerops_site.test", "longitude", "-96.0028"),
					resource.TestCheckResourceAttr("towerops_site.test", "display_order", "7"),
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_version", "2c"),
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_port", "1161"),
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_transport", "udp"),
					resource.TestCheckResourceAttr("towerops_site.test", "parent_site_id", parentSiteID),
					resource.TestCheckResourceAttr("towerops_site.test", "agent_token_id", agentTokenID),
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community_set", "false"),
				),
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
			writeSiteError(w, http.StatusUnprocessableEntity, "validation_error", "name is required")
			return
		}
		writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
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

func TestAccSiteResource_updateError(t *testing.T) {
	var siteID string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			writeSiteData(w, http.StatusCreated, Site{
				ID:         siteID,
				Name:       "Original Name",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       "Original Name",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteError(w, http.StatusInternalServerError, "internal_error", "update failed")

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
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
			writeSiteData(w, http.StatusCreated, Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteError(w, http.StatusInternalServerError, "internal_error", "delete failed")

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
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
	var communityStored bool
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "test-site-id"
			communityStored = decodeSiteRequest(r).SNMPCommunity != nil
			writeSiteData(w, http.StatusCreated, Site{
				ID:               siteID,
				Name:             "SNMP Site",
				SNMPCommunitySet: communityStored,
				InsertedAt:       "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteData(w, http.StatusOK, Site{
				ID:               siteID,
				Name:             "SNMP Site",
				SNMPCommunitySet: communityStored,
				InsertedAt:       "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
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
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community_set", "true"),
				),
			},
		},
	})
}

// TestAccSiteResource_snmpCommunityWriteOnly proves the configured community
// survives every phase even though the API never returns it. The API answers
// with snmp_community_set only, so a provider that copied the response over
// the configured value would either blank the attribute or produce a
// perpetual diff on the second (identical config) step.
func TestAccSiteResource_snmpCommunityWriteOnly(t *testing.T) {
	var siteID string
	var currentName string
	var communityStored bool
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		reply := func(status int) {
			writeSiteData(w, status, Site{
				ID:               siteID,
				Name:             currentName,
				SNMPCommunitySet: communityStored,
				InsertedAt:       "2024-01-01T00:00:00Z",
			})
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			requested := decodeSiteRequest(r)
			siteID = "write-only-site-id"
			currentName = requested.Name
			communityStored = requested.SNMPCommunity != nil
			reply(http.StatusCreated)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			reply(http.StatusOK)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/sites/"+siteID:
			requested := decodeSiteRequest(r)
			currentName = requested.Name
			if requested.SNMPCommunity != nil {
				communityStored = true
			}
			reply(http.StatusOK)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccSiteResourceConfigWithSNMPCommunity(server.URL, "Write Only Site", "s3cret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community", "s3cret"),
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community_set", "true"),
				),
			},
			{
				// Same config: refresh reads the site back without a
				// community and the plan must still be empty.
				Config: testAccSiteResourceConfigWithSNMPCommunity(server.URL, "Write Only Site", "s3cret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community", "s3cret"),
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community_set", "true"),
				),
			},
			{
				// Update path: the community must not be lost when another
				// attribute changes.
				Config: testAccSiteResourceConfigWithSNMPCommunity(server.URL, "Write Only Site Renamed", "s3cret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "name", "Write Only Site Renamed"),
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community", "s3cret"),
					resource.TestCheckResourceAttr("towerops_site.test", "snmp_community_set", "true"),
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

func TestAccSiteResource_recreateOn404(t *testing.T) {
	var siteID string
	var siteDeleted bool
	var currentName string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			siteID = "new-site-id"
			siteDeleted = false
			currentName = decodeSiteRequest(r).Name
			writeSiteData(w, http.StatusCreated, Site{
				ID:         siteID,
				Name:       currentName,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			if siteDeleted {
				writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
				return
			}
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       currentName,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/sites/"+siteID:
			if siteDeleted {
				writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
				return
			}
			currentName = decodeSiteRequest(r).Name
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       currentName,
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			siteDeleted = true
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccSiteResourceConfig(server.URL, "Original Site"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "name", "Original Site"),
				),
			},
			{
				PreConfig: func() {
					mu.Lock()
					siteDeleted = true
					mu.Unlock()
				},
				Config: testAccSiteResourceConfig(server.URL, "Updated Site"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_site.test", "name", "Updated Site"),
					resource.TestCheckResourceAttrSet("towerops_site.test", "id"),
				),
			},
		},
	})
}

func TestAccSiteResource_recreateOn404_createError(t *testing.T) {
	var siteID string
	var siteDeleted bool
	var createCount int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			createCount++
			if createCount > 1 {
				writeSiteError(w, http.StatusUnprocessableEntity, "validation_error", "create failed after 404")
				return
			}
			siteID = "test-site-id"
			siteDeleted = false
			writeSiteData(w, http.StatusCreated, Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sites/"+siteID:
			if siteDeleted {
				writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
				return
			}
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       "Test Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/sites/"+siteID:
			if siteDeleted {
				writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
				return
			}
			writeSiteData(w, http.StatusOK, Site{
				ID:         siteID,
				Name:       "Updated Site",
				InsertedAt: "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sites/"+siteID:
			siteDeleted = true
			writeSiteDeleted(w)

		default:
			writeSiteError(w, http.StatusNotFound, "not_found", "Site not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccSiteResourceConfig(server.URL, "Test Site"),
			},
			{
				PreConfig: func() {
					mu.Lock()
					siteDeleted = true
					mu.Unlock()
				},
				Config:      testAccSiteResourceConfig(server.URL, "Updated Site"),
				ExpectError: regexp.MustCompile(`Failed to create site`),
			},
		},
	})
}
