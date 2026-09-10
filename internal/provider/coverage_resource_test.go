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

// covTestWriteData emits the `{"data": ...}` success envelope every /api/v1
// endpoint returns.
func covTestWriteData(w http.ResponseWriter, status int, c Coverage) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]Coverage{"data": c})
}

// covTestWriteError emits the `{"error": {...}}` failure envelope.
func covTestWriteError(w http.ResponseWriter, status int, code, message string, fields map[string][]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"details": map[string]any{"fields": fields},
		},
	}
	json.NewEncoder(w).Encode(body)
}

// covTestQueued is the shape the API returns right after create: the compute
// job is enqueued and no raster exists yet.
func covTestQueued(id, name string, azimuth float64) Coverage {
	return Coverage{
		ID:             id,
		Name:           name,
		OrganizationID: "org-1",
		SiteID:         "site-1",
		AntennaSlug:    "rf-elements-tp-sh-30",
		FrequencyMHz:   new(5800),
		TxPowerDBm:     new(22.0),
		CableLossDB:    new(0.0),
		SmGainDBi:      new(0.0),
		HeightAGLM:     new(30.0),
		// The API always echoes the schema defaults, including this one.
		HeightAboveRooftopM: new(0.0),
		AzimuthDeg:          new(azimuth),
		DowntiltDeg:         new(0.0),
		RadiusM:             new(6000),
		CellSizeM:           new(20),
		ReceiverHeightM:     new(3.0),
		RxThresholdDBm:      new(-90.0),
		FoliageTuning:       new(0),
		Status:              "queued",
		ProgressPct:         new(0),
		BBox:                &CoverageBBox{},
		InsertedAt:          "2024-01-01T00:00:00Z",
		UpdatedAt:           "2024-01-01T00:00:00Z",
	}
}

// covTestReady is the same coverage once the worker has finished.
func covTestReady(id, name string, azimuth float64) Coverage {
	c := covTestQueued(id, name, azimuth)
	c.Status = "ready"
	c.ProgressPct = new(100)
	c.ComputedAt = new("2024-01-01T00:05:00Z")
	c.PNGPath = new("coverages/" + id + ".png")
	c.RasterPath = new("coverages/" + id + ".tif")
	c.BBox = &CoverageBBox{
		MinLat: new(32.5),
		MaxLat: new(33.25),
		MinLon: new(-96.75),
		MaxLon: new(-96.25),
	}
	c.UpdatedAt = "2024-01-01T00:05:00Z"
	return c
}

func testAccCoverageResourceConfig(apiURL, name string, azimuth float64) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_coverage" "test" {
  name          = %q
  site_id       = "site-1"
  antenna_slug  = "rf-elements-tp-sh-30"
  frequency_mhz = 5800
  tx_power_dbm  = 22.0
  height_agl_m  = 30.0
  azimuth_deg   = %v
  radius_m      = 6000
}
`, apiURL, name, azimuth)
}

func testAccCoverageResourceConfigTxClearance(apiURL string, txClearance float64) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_coverage" "test" {
  name           = "Clearance Sector"
  site_id        = "site-1"
  antenna_slug   = "rf-elements-tp-sh-30"
  frequency_mhz  = 5800
  tx_power_dbm   = 22.0
  height_agl_m   = 30.0
  azimuth_deg    = 0
  radius_m       = 6000
  tx_clearance_m = %v
}
`, apiURL, txClearance)
}

// TestAccCoverageResource_txClearancePersists pins the one input the API
// accepts but never returns: state must keep the configured value instead of
// nulling it on refresh, otherwise every plan would be dirty forever.
func TestAccCoverageResource_txClearancePersists(t *testing.T) {
	const coverageID = "coverage-4"

	var sent *float64
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/coverages":
			var body map[string]Coverage
			json.NewDecoder(r.Body).Decode(&body)
			sent = body["coverage"].TxClearanceM
			// The serializer drops tx_clearance_m from every response.
			covTestWriteData(w, http.StatusCreated, covTestQueued(coverageID, "Clearance Sector", 0))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/coverages/"+coverageID:
			covTestWriteData(w, http.StatusOK, covTestQueued(coverageID, "Clearance Sector", 0))

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/coverages/"+coverageID:
			w.WriteHeader(http.StatusNoContent)

		default:
			covTestWriteError(w, http.StatusNotFound, "not_found", "Coverage not found", nil)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCoverageResourceConfigTxClearance(server.URL, 6.5),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_coverage.test", "tx_clearance_m", "6.5"),
					func(*terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if sent == nil || *sent != 6.5 {
							return fmt.Errorf("tx_clearance_m was not sent to the API: %v", sent)
						}
						return nil
					},
				),
			},
			{
				// A refresh must not clear the value the API never echoes.
				Config:   testAccCoverageResourceConfigTxClearance(server.URL, 6.5),
				PlanOnly: true,
			},
		},
	})
}

// TestAccCoverageResource_asyncLifecycle covers create plus read: create
// records the queued status without polling, and a later refresh picks up the
// terminal status and the computed bbox. The final step proves import.
func TestAccCoverageResource_asyncLifecycle(t *testing.T) {
	const coverageID = "coverage-1"

	var computed bool
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/coverages":
			covTestWriteData(w, http.StatusCreated, covTestQueued(coverageID, "North Sector", 0))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/coverages/"+coverageID:
			if computed {
				covTestWriteData(w, http.StatusOK, covTestReady(coverageID, "North Sector", 0))
				return
			}
			covTestWriteData(w, http.StatusOK, covTestQueued(coverageID, "North Sector", 0))

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/coverages/"+coverageID:
			w.WriteHeader(http.StatusNoContent)

		default:
			covTestWriteError(w, http.StatusNotFound, "not_found", "Coverage not found", nil)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCoverageResourceConfig(server.URL, "North Sector", 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_coverage.test", "id", coverageID),
					resource.TestCheckResourceAttr("towerops_coverage.test", "name", "North Sector"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "organization_id", "org-1"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "status", "queued"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "progress_pct", "0"),
					// Server derived, never configured.
					resource.TestCheckResourceAttr("towerops_coverage.test", "cell_size_m", "20"),
					// Schema defaults survive the round trip.
					resource.TestCheckResourceAttr("towerops_coverage.test", "downtilt_deg", "0"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "receiver_height_m", "3"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "rx_threshold_dbm", "-90"),
				),
			},
			{
				PreConfig: func() {
					mu.Lock()
					computed = true
					mu.Unlock()
				},
				Config: testAccCoverageResourceConfig(server.URL, "North Sector", 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_coverage.test", "status", "ready"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "progress_pct", "100"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "computed_at", "2024-01-01T00:05:00Z"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "bbox.min_lat", "32.5"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "bbox.max_lat", "33.25"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "bbox.min_lon", "-96.75"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "bbox.max_lon", "-96.25"),
				),
			},
			{
				ResourceName:      "towerops_coverage.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCoverageResource_update(t *testing.T) {
	const coverageID = "coverage-2"

	var azimuth = 0.0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/coverages":
			var body map[string]Coverage
			json.NewDecoder(r.Body).Decode(&body)
			if v := body["coverage"].AzimuthDeg; v != nil {
				azimuth = *v
			}
			covTestWriteData(w, http.StatusCreated, covTestQueued(coverageID, "West Sector", azimuth))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/coverages/"+coverageID:
			covTestWriteData(w, http.StatusOK, covTestQueued(coverageID, "West Sector", azimuth))

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/coverages/"+coverageID:
			var body map[string]Coverage
			json.NewDecoder(r.Body).Decode(&body)
			if v := body["coverage"].AzimuthDeg; v != nil {
				azimuth = *v
			}
			// PATCH does not recompute: the status stays where it was.
			covTestWriteData(w, http.StatusOK, covTestQueued(coverageID, "West Sector", azimuth))

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/coverages/"+coverageID:
			w.WriteHeader(http.StatusNoContent)

		default:
			covTestWriteError(w, http.StatusNotFound, "not_found", "Coverage not found", nil)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCoverageResourceConfig(server.URL, "West Sector", 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_coverage.test", "azimuth_deg", "0"),
				),
			},
			{
				Config: testAccCoverageResourceConfig(server.URL, "West Sector", 90),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_coverage.test", "azimuth_deg", "90"),
					resource.TestCheckResourceAttr("towerops_coverage.test", "id", coverageID),
				),
			},
		},
	})
}

func TestAccCoverageResource_validationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/coverages":
			covTestWriteError(w, http.StatusUnprocessableEntity, "validation_error", "Validation failed",
				map[string][]string{"name": {"can't be blank"}})

		default:
			covTestWriteError(w, http.StatusNotFound, "not_found", "Coverage not found", nil)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config:      testAccCoverageResourceConfig(server.URL, "", 0),
				ExpectError: regexp.MustCompile(`name:\s+can't\s+be\s+blank`),
			},
		},
	})
}

func TestAccCoverageResource_readNotFoundRemovesResource(t *testing.T) {
	const coverageID = "coverage-3"

	var gone bool
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/coverages":
			gone = false
			covTestWriteData(w, http.StatusCreated, covTestQueued(coverageID, "Ghost Sector", 0))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/coverages/"+coverageID:
			if gone {
				covTestWriteError(w, http.StatusNotFound, "not_found", "Coverage not found", nil)
				return
			}
			covTestWriteData(w, http.StatusOK, covTestQueued(coverageID, "Ghost Sector", 0))

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/coverages/"+coverageID:
			gone = true
			w.WriteHeader(http.StatusNoContent)

		default:
			covTestWriteError(w, http.StatusNotFound, "not_found", "Coverage not found", nil)
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccCoverageResourceConfig(server.URL, "Ghost Sector", 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("towerops_coverage.test", "id"),
				),
			},
			{
				// Deleted upstream: the refresh drops it from state and the
				// next plan proposes a fresh create.
				PreConfig: func() {
					mu.Lock()
					gone = true
					mu.Unlock()
				},
				Config:             testAccCoverageResourceConfig(server.URL, "Ghost Sector", 0),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
