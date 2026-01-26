---
page_title: "towerops_device Resource - TowerOps"
description: |-
  Manages a TowerOps device.
---

# towerops_device (Resource)

Manages a TowerOps device. Devices represent network equipment at a site.

## Example Usage

```terraform
resource "towerops_device" "router" {
  site_id    = towerops_site.example.id
  name       = "Core Router"
  ip_address = "192.168.1.1"

  monitoring_enabled = true
  snmp_enabled       = true
  snmp_version       = "2c"
  snmp_port          = 161
}
```

### Minimal Configuration

```terraform
resource "towerops_device" "switch" {
  site_id    = towerops_site.example.id
  name       = "Access Switch"
  ip_address = "192.168.1.2"
}
```

## Schema

### Required

- `site_id` (String) - The ID of the site this device belongs to. Changing this forces a new resource.
- `name` (String) - The name of the device.
- `ip_address` (String) - The IP address of the device.

### Optional

- `description` (String) - A description of the device.
- `monitoring_enabled` (Boolean) - Whether monitoring is enabled for this device. Default: `true`.
- `snmp_enabled` (Boolean) - Whether SNMP polling is enabled for this device. Default: `true`.
- `snmp_version` (String) - The SNMP version to use (`1`, `2c`, or `3`). Default: `"2c"`.
- `snmp_port` (Number) - The SNMP port to use. Default: `161`.

### Read-Only

- `id` (String) - The unique identifier of the device.
- `inserted_at` (String) - The timestamp when the device was created.

## Import

Devices can be imported using their UUID:

```shell
terraform import towerops_device.router 7c9e6679-7425-40de-944b-e07fc1f90ae7
```
