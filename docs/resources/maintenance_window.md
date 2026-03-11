---
page_title: "towerops_maintenance_window Resource - TowerOps"
description: |-
  Manages a TowerOps maintenance window.
---

# towerops_maintenance_window (Resource)

Manages a TowerOps maintenance window. Maintenance windows suppress alerts during planned work periods and can be scoped to specific sites or devices.

## Example Usage

### Organization-Wide Maintenance Window

```terraform
resource "towerops_maintenance_window" "network_upgrade" {
  name      = "Network Upgrade"
  reason    = "Upgrading core switches to new firmware"
  starts_at = "2024-03-15T02:00:00Z"
  ends_at   = "2024-03-15T06:00:00Z"
}
```

### Site-Scoped Maintenance Window

```terraform
resource "towerops_maintenance_window" "office_maintenance" {
  name      = "Office Network Maintenance"
  reason    = "Replacing office UPS"
  starts_at = "2024-03-20T22:00:00Z"
  ends_at   = "2024-03-21T02:00:00Z"
  site_id   = towerops_site.main_office.id
}
```

### Device-Scoped Maintenance Window

```terraform
resource "towerops_maintenance_window" "router_update" {
  name             = "Router Firmware Update"
  starts_at        = "2024-03-18T03:00:00Z"
  ends_at          = "2024-03-18T04:00:00Z"
  device_id        = towerops_device.core_router.id
  suppress_alerts  = true
}
```

## Schema

### Required

- `name` (String) - The name of the maintenance window.
- `starts_at` (String) - The start time in ISO 8601 format (e.g. `2024-01-15T02:00:00Z`).
- `ends_at` (String) - The end time in ISO 8601 format (e.g. `2024-01-15T06:00:00Z`).

### Optional

- `reason` (String) - The reason for the maintenance window.
- `suppress_alerts` (Boolean) - Whether to suppress alerts during the window. Defaults to `true`.
- `site_id` (String) - The site to apply the maintenance window to. If omitted, applies to all sites.
- `device_id` (String) - The device to apply the maintenance window to. If omitted, applies to all devices.

### Read-Only

- `id` (String) - The unique identifier of the maintenance window.
- `inserted_at` (String) - The timestamp when the maintenance window was created.

## Import

Maintenance windows can be imported using their UUID:

```shell
terraform import towerops_maintenance_window.example 550e8400-e29b-41d4-a716-446655440000
```
