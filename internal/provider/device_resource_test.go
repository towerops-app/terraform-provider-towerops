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

// deviceRequest decodes the {"device": {...}} payload a provider write sends.
func deviceRequest(r *http.Request) Device {
	var body map[string]Device
	json.NewDecoder(r.Body).Decode(&body)
	return body["device"]
}

// storeDevice mimics what the API persists for a write: the values the request
// carried, the schema defaults for everything it left out, and the write-only
// SNMPv3 passwords reduced to their companion booleans. Echoing the request
// back is the point: a mock that invents values the caller never sent hides
// plan inconsistencies instead of catching them.
func storeDevice(id string, req Device) Device {
	stored := req
	stored.ID = id
	stored.InsertedAt = "2024-01-01T00:00:00Z"

	if stored.Name == nil {
		// The API auto-discovers a name when the request omits one.
		stored.Name = new("Test Device")
	}
	if stored.OrganizationID == nil {
		stored.OrganizationID = new("org-123")
	}
	if stored.MonitoringEnabled == nil {
		stored.MonitoringEnabled = new(true)
	}
	if stored.CheckIntervalSeconds == nil {
		stored.CheckIntervalSeconds = new(300)
	}
	if stored.SNMPEnabled == nil {
		stored.SNMPEnabled = new(true)
	}
	if stored.SNMPVersion == nil {
		stored.SNMPVersion = new("2c")
	}
	if stored.SNMPPort == nil {
		stored.SNMPPort = new(161)
	}
	if stored.DeviceRole == nil {
		stored.DeviceRole = new("other")
	}

	stored.SNMPv3AuthPasswordSet = stored.SNMPv3AuthPassword != nil
	stored.SNMPv3PrivPasswordSet = stored.SNMPv3PrivPassword != nil
	stored.SNMPv3AuthPassword = nil
	stored.SNMPv3PrivPassword = nil

	return stored
}

// deviceCreateView drops the two fields the create response (format_device/1)
// leaves out. Show and update answer with format_device_details/1, which
// carries the full record.
func deviceCreateView(device Device) Device {
	device.Description = nil
	device.CheckIntervalSeconds = nil
	return device
}

// writeDeviceData answers with the {"data": ...} success envelope every
// /api/v1 endpoint returns.
func writeDeviceData(w http.ResponseWriter, status int, device Device) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]Device{"data": device})
}

// writeDeviceDeleted answers with the delete envelope the shared controller
// helper renders.
func writeDeviceDeleted(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"data":{"deleted":true}}`))
}

// writeDeviceError answers with the {"error": {...}} failure envelope.
func writeDeviceError(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"code":%q,"message":%q}}`, code, message)
}

func TestAccDeviceResource_basic(t *testing.T) {
	var mu sync.Mutex
	var stored Device

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			stored = storeDevice("test-device-id", deviceRequest(r))
			writeDeviceData(w, http.StatusCreated, deviceCreateView(stored))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceDeleted(w)

		default:
			writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
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
					// The defaults survive the apply: the provider must keep
					// the planned value rather than the one the API echoes.
					resource.TestCheckResourceAttr("towerops_device.test", "monitoring_enabled", "true"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_enabled", "false"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_version", "2c"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_port", "161"),
					resource.TestCheckResourceAttr("towerops_device.test", "check_interval_seconds", "300"),
					resource.TestCheckResourceAttr("towerops_device.test", "device_role", "other"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmpv3_auth_password_set", "false"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmpv3_priv_password_set", "false"),
					resource.TestCheckResourceAttrSet("towerops_device.test", "id"),
					resource.TestCheckResourceAttrSet("towerops_device.test", "inserted_at"),
				),
			},
		},
	})
}

func TestAccDeviceResource_withAllAttributes(t *testing.T) {
	var mu sync.Mutex
	var stored Device

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			stored = storeDevice("test-device-id", deviceRequest(r))
			writeDeviceData(w, http.StatusCreated, deviceCreateView(stored))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceDeleted(w)

		default:
			writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
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
					// The create response omits description and
					// check_interval_seconds, so the configured values must
					// stay put instead of falling back to the zero value.
					resource.TestCheckResourceAttr("towerops_device.test", "description", "Test description"),
					resource.TestCheckResourceAttr("towerops_device.test", "check_interval_seconds", "600"),
					resource.TestCheckResourceAttr("towerops_device.test", "monitoring_enabled", "true"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_enabled", "true"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_version", "2c"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_port", "161"),
					resource.TestCheckResourceAttr("towerops_device.test", "device_role", "router"),
				),
			},
		},
	})
}

// TestAccDeviceResource_writeOnlySNMPv3Passwords pins the write-only contract:
// the API takes the two SNMPv3 passwords but never returns them, so the
// configured values must stay in state and the *_set booleans must report
// that the API is holding them.
func TestAccDeviceResource_writeOnlySNMPv3Passwords(t *testing.T) {
	var mu sync.Mutex
	var stored Device

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			stored = storeDevice("test-device-id", deviceRequest(r))
			writeDeviceData(w, http.StatusCreated, deviceCreateView(stored))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceDeleted(w)

		default:
			writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
		}
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceResourceConfigSNMPv3(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_version", "3"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmpv3_security_level", "authPriv"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmpv3_username", "snmpuser"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmpv3_auth_password", "auth-secret"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmpv3_priv_password", "priv-secret"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmpv3_auth_password_set", "true"),
					resource.TestCheckResourceAttr("towerops_device.test", "snmpv3_priv_password_set", "true"),
				),
			},
		},
	})
}

func TestAccDeviceResource_update(t *testing.T) {
	var mu sync.Mutex
	var stored Device

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			stored = storeDevice("test-device-id", deviceRequest(r))
			writeDeviceData(w, http.StatusCreated, deviceCreateView(stored))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/devices/"+stored.ID:
			stored = storeDevice(stored.ID, deviceRequest(r))
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceDeleted(w)

		default:
			writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
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
					resource.TestCheckResourceAttr("towerops_device.test", "snmp_enabled", "false"),
				),
			},
		},
	})
}

func TestAccDeviceResource_recreateOn404(t *testing.T) {
	var mu sync.Mutex
	var stored Device
	var deviceDeleted bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			stored = storeDevice("test-device-id", deviceRequest(r))
			deviceDeleted = false
			writeDeviceData(w, http.StatusCreated, deviceCreateView(stored))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+stored.ID:
			if deviceDeleted {
				writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
				return
			}
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/devices/"+stored.ID:
			if deviceDeleted {
				writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
				return
			}
			stored = storeDevice(stored.ID, deviceRequest(r))
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+stored.ID:
			deviceDeleted = true
			writeDeviceDeleted(w)

		default:
			writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
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
	var mu sync.Mutex
	var stored Device

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			stored = storeDevice("imported-device-id", deviceRequest(r))
			writeDeviceData(w, http.StatusCreated, deviceCreateView(stored))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/imported-device-id":
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/imported-device-id":
			writeDeviceDeleted(w)

		default:
			writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
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
  site_id                = "site-123"
  name                   = "Full Device"
  ip_address             = "10.0.0.1"
  description            = "Test description"
  device_role            = "router"
  monitoring_enabled     = true
  check_interval_seconds = 600
  snmp_enabled           = true
  snmp_version           = "2c"
  snmp_port              = 161
}
`, apiURL)
}

func testAccDeviceResourceConfigSNMPv3(apiURL string) string {
	return fmt.Sprintf(`
provider "towerops" {
  token   = "test-token"
  api_url = %q
}

resource "towerops_device" "test" {
  site_id      = "site-123"
  name         = "Secure Switch"
  ip_address   = "192.168.1.10"
  snmp_enabled = true
  snmp_version = "3"

  snmpv3_security_level = "authPriv"
  snmpv3_username       = "snmpuser"
  snmpv3_auth_protocol  = "SHA-256"
  snmpv3_auth_password  = "auth-secret"
  snmpv3_priv_protocol  = "AES"
  snmpv3_priv_password  = "priv-secret"
}
`, apiURL)
}

func TestAccDeviceResource_createError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices" {
			writeDeviceError(w, http.StatusBadRequest, "bad_request", "ip_address is required")
			return
		}
		writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
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
	var mu sync.Mutex
	var stored Device

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			stored = storeDevice("test-device-id", deviceRequest(r))
			writeDeviceData(w, http.StatusCreated, deviceCreateView(stored))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceError(w, http.StatusInternalServerError, "internal_error", "update failed")

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceDeleted(w)

		default:
			writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
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
	var mu sync.Mutex
	var stored Device

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			stored = storeDevice("test-device-id", deviceRequest(r))
			writeDeviceData(w, http.StatusCreated, deviceCreateView(stored))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+stored.ID:
			writeDeviceError(w, http.StatusInternalServerError, "internal_error", "delete failed")

		default:
			writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
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
	var mu sync.Mutex
	var stored Device
	var deviceDeleted bool
	var createCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/devices":
			createCount++
			if createCount > 1 {
				writeDeviceError(w, http.StatusBadRequest, "bad_request", "create failed after 404")
				return
			}
			stored = storeDevice("test-device-id", deviceRequest(r))
			deviceDeleted = false
			writeDeviceData(w, http.StatusCreated, deviceCreateView(stored))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/devices/"+stored.ID:
			if deviceDeleted {
				writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
				return
			}
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/devices/"+stored.ID:
			if deviceDeleted {
				writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
				return
			}
			stored = storeDevice(stored.ID, deviceRequest(r))
			writeDeviceData(w, http.StatusOK, stored)

		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/devices/"+stored.ID:
			deviceDeleted = true
			writeDeviceDeleted(w)

		default:
			writeDeviceError(w, http.StatusNotFound, "not_found", "Device not found")
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
