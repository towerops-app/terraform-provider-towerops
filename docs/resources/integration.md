---
page_title: "towerops_integration Resource - TowerOps"
description: |-
  Manages a TowerOps integration with third-party services.
---

# towerops_integration (Resource)

Manages a TowerOps integration with a third-party system such as PagerDuty, NetBox, Sonar, Splynx, or a MikroTik controller.

~> **Credentials cannot be set through this resource.** The integrations
endpoint filters request attributes through an allowlist of `provider`,
`enabled`, `config`, `api_key`, `api_url`, and `sync_interval_minutes`, and
`Integration.changeset/2` then casts only `provider`, `enabled`, `settings`,
and `sync_interval_minutes`. Every credential key is dropped somewhere along
that path, so `provider_type`, `enabled`, and `sync_interval_minutes` are the
entire settable surface over REST today. Configure the integration's
credentials in the TowerOps UI, then manage its lifecycle here.

## Example Usage

### PagerDuty

```terraform
resource "towerops_integration" "pagerduty" {
  provider_type = "pagerduty"
  enabled       = true
}
```

### NetBox with a sync interval

```terraform
resource "towerops_integration" "netbox" {
  provider_type         = "netbox"
  enabled               = true
  sync_interval_minutes = 15
}
```

### Registered but paused

```terraform
resource "towerops_integration" "splynx" {
  provider_type = "splynx"
  enabled       = false
}
```

## Schema

### Required

- `provider_type` (String) - The integration provider. The API accepts exactly `preseem`, `gaiia`, `pagerduty`, `netbox`, `sonar`, `splynx`, `visp`, `uisp`, `cn_maestro`, and `mikrotik`; anything else is rejected with a `validation_error`. An organization may have only one integration per provider.

### Optional

- `enabled` (Boolean) - Whether the integration is enabled. Defaults to `true` in this provider, which always sends the value; the API's own default for an omitted field is `false`.
- `sync_interval_minutes` (Number) - How often the integration syncs, in minutes. Must be greater than zero. The server default is 10.

### Read-Only

- `id` (String) - The unique identifier of the integration.
- `last_synced_at` (String) - The timestamp of the integration's last sync attempt. Null until the integration has synced.
- `last_sync_status` (String) - The outcome of the integration's last sync attempt.
- `last_sync_message` (String) - Detail reported by the integration's last sync attempt, such as an upstream error.
- `inserted_at` (String) - The timestamp when the integration was created.
- `updated_at` (String) - The timestamp when the integration was last modified.

## Import

Integrations can be imported using their UUID:

```shell
terraform import towerops_integration.example 550e8400-e29b-41d4-a716-446655440000
```
