---
page_title: "towerops_device Resource - TowerOps"
description: |-
  Manages a TowerOps device.
---

# towerops_device (Resource)

Manages a TowerOps device. Devices represent network equipment that can be assigned to a site or directly to the organization.

## Example Usage

### Site-based Device with SNMP v2c

```terraform
resource "towerops_device" "router" {
  site_id    = towerops_site.example.id
  name       = "Core Router"
  ip_address = "192.168.1.1"

  monitoring_enabled = true
  # 300 seconds is the floor the API enforces, 3600 the ceiling.
  check_interval_seconds = 300
  snmp_enabled           = true
  snmp_version           = "2c"
  snmp_port              = 161
}
```

### Organization-level Device (Site-less)

Devices can be created without a site, assigned directly to the organization:

```terraform
resource "towerops_device" "cloud_router" {
  name       = "Cloud Router"
  ip_address = "10.0.1.1"
  # No site_id - device belongs directly to organization
}
```

### Device with SNMPv3

```terraform
resource "towerops_device" "secure_switch" {
  name         = "Secure Switch"
  ip_address   = "192.168.1.10"
  snmp_version = "3"

  # SNMPv3 security settings
  snmpv3_security_level = "authPriv"
  snmpv3_username       = "snmpuser"
  snmpv3_auth_protocol  = "SHA-256"
  snmpv3_auth_password  = var.snmp_auth_password
  snmpv3_priv_protocol  = "AES"
  snmpv3_priv_password  = var.snmp_priv_password
}
```

### Ping/DNS-only Device (no SNMP)

```terraform
resource "towerops_device" "resolver" {
  site_id    = towerops_site.example.id
  name       = "resolver1"
  ip_address = "204.110.191.240"
}
```

### Minimal Configuration

```terraform
resource "towerops_device" "switch" {
  name       = "Access Switch"
  ip_address = "192.168.1.2"
}
```

## Schema

### Required

- `ip_address` (String) - The IP address of the device.

### Optional

- `site_id` (String) - The ID of the site this device belongs to. Optional if `organization_id` is provided. Changing this forces a new resource.
- `organization_id` (String) - The ID of the organization this device belongs to. Defaults to the authenticated organization if not provided. Changing this forces a new resource.
- `name` (String) - The name of the device. If not provided, will be auto-discovered from SNMP.
- `description` (String) - A description of the device.
- `device_role` (String) - The device role. Must be one of: `router`, `switch`, `access_point`, `backhaul`, `server`, `other`. Defaults to `other`.
- `monitoring_enabled` (Boolean) - Whether monitoring is enabled for this device. Default: `true`.
- `check_interval_seconds` (Number) - How often the device is checked, in seconds. The API accepts `300` to `3600` inclusive and rejects anything outside that range. Default: `300`.
- `snmp_enabled` (Boolean) - Whether SNMP polling is enabled for this device. Default: `false`.
- `snmp_version` (String) - The SNMP version to use (`1`, `2c`, or `3`). Default: `"2c"`.
- `snmp_port` (Number) - The SNMP port to use. Default: `161`.

#### SNMPv3 Fields (only used when `snmp_version = "3"`)

- `snmpv3_security_level` (String) - SNMPv3 security level. Must be one of:
  - `noAuthNoPriv` - No authentication or privacy
  - `authNoPriv` - Authentication without privacy
  - `authPriv` - Authentication with privacy
- `snmpv3_username` (String) - SNMPv3 username.
- `snmpv3_auth_protocol` (String) - SNMPv3 authentication protocol. Must be one of: `MD5`, `SHA`, `SHA-224`, `SHA-256`, `SHA-384`, `SHA-512`.
- `snmpv3_auth_password` (String, Sensitive, write-only) - SNMPv3 authentication password.
- `snmpv3_priv_protocol` (String) - SNMPv3 privacy protocol. Must be one of: `DES`, `AES`, `AES-192`, `AES-256`.
- `snmpv3_priv_password` (String, Sensitive, write-only) - SNMPv3 privacy password.

The two SNMPv3 passwords are write-only: the API accepts them on create and
update but never returns them, so their state always mirrors the
configuration. Use `snmpv3_auth_password_set` and `snmpv3_priv_password_set`
to see whether the API is currently holding a password.

### Read-Only

- `id` (String) - The unique identifier of the device.
- `snmpv3_auth_password_set` (Boolean) - Whether the API currently holds an SNMPv3 authentication password for this device.
- `snmpv3_priv_password_set` (Boolean) - Whether the API currently holds an SNMPv3 privacy password for this device.
- `inserted_at` (String) - The timestamp when the device was created.

## Import

Devices can be imported using their UUID:

```shell
terraform import towerops_device.router 7c9e6679-7425-40de-944b-e07fc1f90ae7
```
