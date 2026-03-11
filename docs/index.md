---
page_title: "TowerOps Provider"
description: |-
  The TowerOps provider allows you to manage TowerOps resources such as sites, devices, on-call schedules, and escalation policies.
---

# TowerOps Provider

The TowerOps provider allows you to manage [TowerOps](https://towerops.net) resources via Terraform.

## Authentication

The provider requires an API token for authentication. Generate a token from the TowerOps web application under Settings → API Tokens. The token determines which organization's resources are accessible.

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
  name     = "Main Office"
  location = "New York, NY"
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

### Integration

```terraform
resource "towerops_integration" "pagerduty" {
  provider = "pagerduty"
  enabled  = true
}
```

### Maintenance Window

```terraform
resource "towerops_maintenance_window" "network_upgrade" {
  name      = "Network Upgrade"
  reason    = "Upgrading core switches to new firmware"
  starts_at = "2024-03-15T02:00:00Z"
  ends_at   = "2024-03-15T06:00:00Z"
}
```

## Schema

### Required

- `token` (String, Sensitive) - The API token for authenticating with TowerOps.

### Optional

- `api_url` (String) - The base URL for the TowerOps API. Defaults to `https://towerops.net`.
