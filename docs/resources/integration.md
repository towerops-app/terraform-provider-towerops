---
page_title: "towerops_integration Resource - TowerOps"
description: |-
  Manages a TowerOps integration with third-party services.
---

# towerops_integration (Resource)

Manages a TowerOps integration with third-party services such as PagerDuty, Slack, or webhooks.

## Example Usage

### PagerDuty Integration

```terraform
resource "towerops_integration" "pagerduty" {
  provider = "pagerduty"
  enabled  = true
}
```

### Webhook Integration with Sync Interval

```terraform
resource "towerops_integration" "webhook" {
  provider             = "webhook"
  enabled              = true
  sync_interval_minutes = 15
}
```

### Disabled Integration

```terraform
resource "towerops_integration" "slack" {
  provider = "slack"
  enabled  = false
}
```

## Schema

### Required

- `provider` (String) - The integration provider type (e.g. `pagerduty`, `slack`, `webhook`).

### Optional

- `enabled` (Boolean) - Whether the integration is enabled. Defaults to `true`.
- `sync_interval_minutes` (Number) - How often the integration syncs, in minutes.

### Read-Only

- `id` (String) - The unique identifier of the integration.
- `inserted_at` (String) - The timestamp when the integration was created.

## Import

Integrations can be imported using their UUID:

```shell
terraform import towerops_integration.example 550e8400-e29b-41d4-a716-446655440000
```
