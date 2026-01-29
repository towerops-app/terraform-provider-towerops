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

func TestAccDeviceResource_basic(t *testing.T) {
	var deviceID string
	var mu sync.Mutex
	monitoringEnabled := true
	snmpEnabled := true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			deviceID = "test-device-id"
			name := "Test Device"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+deviceID:
			name := "Test Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+deviceID:
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
				Config: testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_device.test", "site_id", "site-123"),
					resource.TestCheckResourceAttr("towerops_device.test", "ip_address", "192.168.1.1"),
					resource.TestCheckResourceAttrSet("towerops_device.test", "id"),
					resource.TestCheckResourceAttrSet("towerops_device.test", "inserted_at"),
				),
			},
		},
	})
}

func TestAccDeviceResource_withAllAttributes(t *testing.T) {
	var deviceID string
	var mu sync.Mutex
	monitoringEnabled := true
	snmpEnabled := true
	snmpVersion := "2c"
	snmpPort := 161
	description := "Test description"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			deviceID = "test-device-id"
			name := "Full Device"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "10.0.0.1",
				Description:       &description,
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+deviceID:
			name := "Full Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "10.0.0.1",
				Description:       &description,
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+deviceID:
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
				Config: testAccDeviceResourceConfigFull(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_device.test", "site_id", "site-123"),
					resource.TestCheckResourceAttr("towerops_device.test", "name", "Full Device"),
					resource.TestCheckResourceAttr("towerops_device.test", "ip_address", "10.0.0.1"),
					resource.TestCheckResourceAttr("towerops_device.test", "description", "Test description"),
					resource.TestCheckResourceAttr("towerops_device.test", "monitoring_enabled", "true"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_enabled", "true"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_version", "2c"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_port", "161"),
				),
			},
		},
	})
}

func TestAccDeviceResource_update(t *testing.T) {
	var deviceID string
	var currentIP string
	var mu sync.Mutex
	monitoringEnabled := true
	snmpEnabled := true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			deviceID = "test-device-id"
			currentIP = "192.168.1.1"
			name := "Test Device"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         currentIP,
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+deviceID:
			name := "Test Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         currentIP,
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/devices/"+deviceID:
			var body map[string]Device
			json.NewDecoder(r.Body).Decode(&body)
			currentIP = body["device"].IPAddress
			name := "Test Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         currentIP,
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+deviceID:
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
				Config: testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_device.test", "ip_address", "192.168.1.1"),
				),
			},
			{
				Config: testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_device.test", "ip_address", "192.168.1.2"),
				),
			},
		},
	})
}

func TestAccDeviceResource_recreateOn404(t *testing.T) {
	var deviceID string
	var deviceDeleted bool
	var currentIP string
	var mu sync.Mutex
	monitoringEnabled := true
	snmpEnabled := true
	snmpVersion := "2c"
	snmpPort := 161

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			var body map[string]Device
			json.NewDecoder(r.Body).Decode(&body)
			deviceID = "new-device-id"
			deviceDeleted = false
			currentIP = body["device"].IPAddress
			name := "Auto-discovered Device"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         currentIP,
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+deviceID:
			if deviceDeleted {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "device not found"}`))
				return
			}
			name := "Auto-discovered Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         currentIP,
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/devices/"+deviceID:
			if deviceDeleted {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "device not found"}`))
				return
			}
			var body map[string]Device
			json.NewDecoder(r.Body).Decode(&body)
			currentIP = body["device"].IPAddress
			name := "Auto-discovered Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         currentIP,
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+deviceID:
			deviceDeleted = true
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
				Config: testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_device.test", "ip_address", "192.168.1.1"),
				),
			},
			{
				PreConfig: func() {
					mu.Lock()
					deviceDeleted = true
					mu.Unlock()
				},
				Config: testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_device.test", "ip_address", "192.168.1.2"),
					resource.TestCheckResourceAttrSet("towerops_device.test", "id"),
				),
			},
		},
	})
}

func TestAccDeviceResource_importState(t *testing.T) {
	var deviceID string
	var mu sync.Mutex
	monitoringEnabled := true
	snmpEnabled := true
	snmpVersion := "2c"
	snmpPort := 161

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			deviceID = "imported-device-id"
			name := "Imported Device"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/imported-device-id":
			name := "Imported Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                "imported-device-id",
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/imported-device-id":
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
				Config: testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.1"),
			},
			{
				ResourceName:      "towerops_device.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDeviceResourceConfig(apiURL, siteID, ipAddress string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_device" "test" {
  site_id    = %q
  ip_address = %q
}
`, apiURL, siteID, ipAddress)
}

func testAccDeviceResourceConfigFull(apiURL string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_device" "test" {
  site_id            = "site-123"
  name               = "Full Device"
  ip_address         = "10.0.0.1"
  description        = "Test description"
  monitoring_enabled = true
  snmp_enabled       = true
  snmp_version       = "2c"
  snmp_port          = 161
}
`, apiURL)
}

func TestAccDeviceResource_createError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "ip_address is required"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config:      testAccDeviceResourceConfig(server.URL, "site-123", "invalid"),
				ExpectError: regexp.MustCompile(`Failed to create device`),
			},
		},
	})
}

func TestAccDeviceResource_updateError(t *testing.T) {
	var deviceID string
	var mu sync.Mutex
	monitoringEnabled := true
	snmpEnabled := true
	snmpVersion := "2c"
	snmpPort := 161

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			deviceID = "test-device-id"
			name := "Test Device"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+deviceID:
			name := "Test Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/devices/"+deviceID:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "update failed"}`))

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+deviceID:
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
				Config: testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.1"),
			},
			{
				Config:      testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.2"),
				ExpectError: regexp.MustCompile(`Failed to update device`),
			},
		},
	})
}

func TestAccDeviceResource_deleteError(t *testing.T) {
	var deviceID string
	var mu sync.Mutex
	monitoringEnabled := true
	snmpEnabled := true
	snmpVersion := "2c"
	snmpPort := 161

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			deviceID = "test-device-id"
			name := "Test Device"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+deviceID:
			name := "Test Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+deviceID:
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
				Config:  testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.1"),
				Destroy: true,
			},
		},
		ErrorCheck: func(err error) error {
			if err != nil && regexp.MustCompile(`Failed to delete device`).MatchString(err.Error()) {
				return nil
			}
			return err
		},
	})
}

func TestAccDeviceResource_recreateOn404_createError(t *testing.T) {
	var deviceID string
	var deviceDeleted bool
	var createCount int
	var mu sync.Mutex
	monitoringEnabled := true
	snmpEnabled := true
	snmpVersion := "2c"
	snmpPort := 161

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			createCount++
			if createCount > 1 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "create failed after 404"}`))
				return
			}
			deviceID = "test-device-id"
			deviceDeleted = false
			name := "Test Device"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+deviceID:
			if deviceDeleted {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "device not found"}`))
				return
			}
			name := "Test Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.1",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/devices/"+deviceID:
			if deviceDeleted {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "device not found"}`))
				return
			}
			name := "Test Device"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Device{
				ID:                deviceID,
				SiteID:            "site-123",
				Name:              &name,
				IPAddress:         "192.168.1.2",
				MonitoringEnabled: &monitoringEnabled,
				SNMPEnabled:       &snmpEnabled,
				SNMPVersion:       &snmpVersion,
				SNMPPort:          &snmpPort,
				InsertedAt:        "2024-01-01T00:00:00Z",
			})

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+deviceID:
			deviceDeleted = true
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
				Config: testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.1"),
			},
			{
				PreConfig: func() {
					mu.Lock()
					deviceDeleted = true
					mu.Unlock()
				},
				Config:      testAccDeviceResourceConfig(server.URL, "site-123", "192.168.1.2"),
				ExpectError: regexp.MustCompile(`Failed to create device`),
			},
		},
	})
}
