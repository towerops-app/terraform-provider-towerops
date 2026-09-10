---
page_title: "towerops_site Resource - TowerOps"
description: |-
  Manages a TowerOps site.
---

# towerops_site (Resource)

Manages a TowerOps site. Sites represent physical locations that contain devices.

A site can carry SNMP defaults that its devices inherit, and it can be nested
under another site through `parent_site_id`.

## Example Usage

### Basic

```terraform
resource "towerops_site" "example" {
  name           = "Main Office"
  location       = "New York, NY"
  snmp_community = "public"
}
```

### With Geographic Coordinates

```terraform
resource "towerops_site" "tower_site" {
  name      = "Verona Tower"
  location  = "Verona, TX"
  address   = "123 Main St, Verona, TX 75482"
  latitude  = 33.4356
  longitude = -96.0028
}
```

### With SNMP Defaults and a Parent Site

```terraform
resource "towerops_site" "region" {
  name = "North Region"
}

resource "towerops_site" "tower" {
  name           = "Verona Tower"
  description    = "Backhaul tower serving the north region"
  parent_site_id = towerops_site.region.id
  display_order  = 10

  snmp_version   = "2c"
  snmp_community = "public"
  snmp_port      = 161
  snmp_transport = "udp"
}
```

## Schema

### Required

- `name` (String) - The name of the site. Must be between 2 and 200 characters.

### Optional

- `description` (String) - A longer description of the site. Maximum 1000 characters.
- `location` (String) - A short description of the physical location. Maximum 200 characters.
- `address` (String) - The street address of the site. Maximum 500 characters.
- `latitude` (Float, Computed) - The latitude of the site. Must be between -90 and 90.
- `longitude` (Float, Computed) - The longitude of the site. Must be between -180 and 180.
- `display_order` (Number, Computed) - The sort position of the site in TowerOps listings.
- `snmp_community` (String, Sensitive, write-only) - The default SNMP community string for devices at this site. See "Write-only attributes" below.
- `snmp_version` (String, Computed) - The default SNMP version for devices at this site. One of `"1"`, `"2c"`, or `"3"`.
- `snmp_port` (Number, Computed) - The default SNMP port for devices at this site. Must be between 1 and 65535.
- `snmp_transport` (String, Computed) - The default SNMP transport for devices at this site, for example `udp`.
- `agent_token_id` (String, Computed) - The ID of the agent token that polls this site. Assigned by TowerOps when it is not configured.
- `parent_site_id` (String) - The ID of the parent site, for nesting sites into a hierarchy. A site may not be its own ancestor.

Attributes marked `Computed` above are optional in configuration; when they are
omitted, the value TowerOps assigns is written to state.

### Read-Only

- `id` (String) - The unique identifier of the site.
- `snmp_community_set` (Boolean) - Whether the site has an SNMP community string stored.
- `inserted_at` (String) - The timestamp when the site was created.

## Write-only attributes

The API accepts `snmp_community` on create and update but never returns it.
The provider therefore keeps the configured value in state and never overwrites
it from an API response. Use the computed `snmp_community_set` attribute to see
whether TowerOps currently holds a community string for the site:

```terraform
output "site_has_community" {
  value = towerops_site.tower.snmp_community_set
}
```

Because the value is never read back, a community string that is changed
outside of Terraform is not detected as drift. Removing `snmp_community` from
the configuration stops the provider from sending it, but does not clear the
stored value.

## Import

Sites can be imported using their UUID:

```shell
terraform import towerops_site.example 550e8400-e29b-41d4-a716-446655440000
```

`snmp_community` cannot be imported because the API does not return it. After
an import, add the community to your configuration; the next apply sends it to
the API.
