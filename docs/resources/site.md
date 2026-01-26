---
page_title: "towerops_site Resource - TowerOps"
description: |-
  Manages a TowerOps site.
---

# towerops_site (Resource)

Manages a TowerOps site. Sites represent physical locations that contain devices.

## Example Usage

```terraform
resource "towerops_site" "example" {
  name           = "Main Office"
  location       = "New York, NY"
  snmp_community = "public"
}
```

## Schema

### Required

- `name` (String) - The name of the site. Must be between 2 and 200 characters.

### Optional

- `location` (String) - The physical location or address of the site.
- `snmp_community` (String, Sensitive) - The default SNMP community string for devices at this site.

### Read-Only

- `id` (String) - The unique identifier of the site.
- `inserted_at` (String) - The timestamp when the site was created.

## Import

Sites can be imported using their UUID:

```shell
terraform import towerops_site.example 550e8400-e29b-41d4-a716-446655440000
```
