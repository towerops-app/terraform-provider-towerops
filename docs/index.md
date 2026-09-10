---
page_title: "TowerOps Provider"
description: |-
  The TowerOps provider allows you to manage TowerOps resources such as sites, devices, on-call schedules, and escalation policies.
---

# TowerOps Provider

The TowerOps provider allows you to manage [TowerOps](https://towerops.net) resources via Terraform.

## Authentication

The provider requires an API token for authentication. Generate a token from the TowerOps web application under Settings, API Tokens. The token determines which organization's resources are accessible.

### Token scopes

TowerOps API tokens are scoped. A token that is missing the scope an endpoint
requires is rejected with HTTP 403 and the machine-readable code
`insufficient_scope`, which the provider surfaces as
`API error (403 insufficient_scope): ...`. Grant the token every scope the
resources in your configuration need:

| Resource | Scopes |
| --- | --- |
| `towerops_organization` | `organization:read`, `organization:write` |
| `towerops_site` | `sites:read`, `sites:write` |
| `towerops_device` | `devices:read`, `devices:write` |
| `towerops_check` | `checks:read`, `checks:write` |
| `towerops_agent` | `agents:read`, `agents:write` |
| `towerops_integration` | `integrations:read`, `integrations:write` |
| `towerops_coverage` | `coverages:read`, `coverages:write` |
| `towerops_schedule` | `config:read`, `config:write` |
| `towerops_escalation_policy` | `config:read`, `config:write` |
| `towerops_maintenance_window` | `config:read`, `config:write` |
| `towerops_webhook_endpoint` | `webhooks:manage` |

Tokens issued before scoping was introduced behave as full-access tokens.

### Error reporting

Every `/api/v1` failure carries a stable machine-readable code alongside its
human-readable message, and the provider includes both in its diagnostics.
The codes are `authentication_required`, `invalid_token`, `forbidden`,
`insufficient_scope`, `not_found`, `bad_request`, `validation_error`,
`conflict`, `upstream_error`, `timeout`, `rate_limited`, and
`internal_error`. A `validation_error` also lists the offending fields, so a
rejected apply reports for example
`API error (422 validation_error): Validation failed (name: can't be blank)`.

## Example Usage

### Basic Usage with Site Hierarchy

```terraform
terraform {
  required_providers {
    towerops = {
      source  = "towerops/towerops"
      version = "~> 0.1"
    }
  }
}

provider "towerops" {
  token = var.towerops_api_token
}

resource "towerops_site" "example" {
  name      = "Main Office"
  location  = "New York, NY"
  address   = "350 5th Ave, New York, NY 10118"
  latitude  = 40.7484
  longitude = -73.9857
}

resource "towerops_device" "router" {
  site_id    = towerops_site.example.id
  name       = "Core Router"
  ip_address = "192.168.1.1"
}
```

### Site-less Device (Direct Organization Assignment)

Devices can be created without a site, assigned directly to the organization:

```terraform
resource "towerops_device" "cloud_router" {
  name       = "Cloud Router"
  ip_address = "10.0.1.1"
  # No site_id - device belongs directly to organization
}
```

### SNMPv3 Device Configuration

```terraform
resource "towerops_device" "secure_switch" {
  name         = "Secure Switch"
  ip_address   = "192.168.1.10"
  snmp_version = "3"

  # SNMPv3 Authentication and Privacy
  snmpv3_security_level = "authPriv"
  snmpv3_username       = "snmpuser"
  snmpv3_auth_protocol  = "SHA-256"
  snmpv3_auth_password  = var.snmp_auth_password
  snmpv3_priv_protocol  = "AES"
  snmpv3_priv_password  = var.snmp_priv_password
}
```

### Complete Example with Multiple Configurations

```terraform
# Traditional site-based device with SNMP v2c
resource "towerops_device" "legacy_router" {
  site_id      = towerops_site.example.id
  name         = "Legacy Router"
  ip_address   = "192.168.1.1"
  snmp_version = "2c"
}

# Organization-level device with SNMPv3
resource "towerops_device" "modern_switch" {
  name         = "Modern Switch"
  ip_address   = "10.0.2.1"
  snmp_version = "3"

  snmpv3_security_level = "authPriv"
  snmpv3_username       = "admin"
  snmpv3_auth_protocol  = "SHA-256"
  snmpv3_auth_password  = var.snmp_auth_pass
  snmpv3_priv_protocol  = "AES-256"
  snmpv3_priv_password  = var.snmp_priv_pass

  monitoring_enabled = true
  snmp_enabled       = true
}
```

### On-Call Schedule and Escalation Policy

```terraform
resource "towerops_schedule" "primary" {
  name        = "Primary On-Call"
  timezone    = "America/Chicago"
  description = "Main engineering on-call rotation"
}

resource "towerops_escalation_policy" "critical" {
  name         = "Critical Alerts"
  description  = "Escalation for P1 incidents"
  repeat_count = 5
}
```

### Agent Token

```terraform
resource "towerops_agent" "office" {
  name = "Office Poller"
}

output "agent_token" {
  value     = towerops_agent.office.token
  sensitive = true
}
```

### Service Checks

```terraform
resource "towerops_check" "web_health" {
  name            = "Web Health Check"
  check_type      = "http"
  url             = "https://example.com/health"
  expected_status = 200
  content_match   = "\"status\":\"ok\""
}

resource "towerops_check" "gateway_ping" {
  name       = "Gateway Reachability"
  check_type = "ping"
  host       = "10.0.0.1"
}
```

### Integration

```terraform
resource "towerops_integration" "pagerduty" {
  provider_type = "pagerduty"
  enabled       = true
}
```

### Maintenance Window

```terraform
resource "towerops_maintenance_window" "network_upgrade" {
  name      = "Network Upgrade"
  reason    = "Upgrading core switches to new firmware"
  starts_at = "2026-03-15T02:00:00Z"
  ends_at   = "2026-03-15T06:00:00Z"
}

resource "towerops_maintenance_window" "weekly_patching" {
  name            = "Weekly Patching"
  starts_at       = "2026-03-01T03:00:00Z"
  ends_at         = "2026-03-01T05:00:00Z"
  recurring       = true
  recurrence_rule = "FREQ=WEEKLY;BYDAY=SU"
}
```

### RF Coverage

Creating a coverage enqueues an asynchronous compute job, so `status` is
`queued` immediately after apply and becomes `ready` or `failed` once the
worker finishes. Inputs are SI only.

```terraform
resource "towerops_coverage" "north_sector" {
  name          = "North sector 5 GHz"
  site_id       = towerops_site.example.id
  antenna_slug  = "rf-elements-tp-sh-30"
  frequency_mhz = 5800
  tx_power_dbm  = 22.0
  height_agl_m  = 30.0
  azimuth_deg   = 0
  radius_m      = 6437
}
```

### Outbound Webhook

Requires a token with the `webhooks:manage` scope. The signing secret is
returned exactly once, on create.

```terraform
resource "towerops_webhook_endpoint" "alerts" {
  name = "Alert bridge"
  url  = "https://hooks.example.com/towerops/alerts"
  events = [
    "alert.triggered",
    "alert.resolved",
  ]
}

output "webhook_secret" {
  value     = towerops_webhook_endpoint.alerts.secret
  sensitive = true
}
```

## Schema

### Required

- `token` (String, Sensitive) - The API token for authenticating with TowerOps.

### Optional

- `api_url` (String) - The base URL for the TowerOps API. Defaults to `https://towerops.net`.
