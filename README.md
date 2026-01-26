# TowerOps Terraform Provider

Terraform provider for managing [TowerOps](https://towerops.net) resources.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (to build the provider)

## Installation

```hcl
terraform {
  required_providers {
    towerops = {
      source  = "towerops/towerops"
      version = "~> 0.1"
    }
  }
}
```

## Authentication

The provider requires an API token for authentication. Generate a token from the TowerOps web application under Settings → API Tokens.

```hcl
provider "towerops" {
  token = var.towerops_api_token
}
```

You can also set the token via environment variable:

```bash
export TOWEROPS_TOKEN="your-api-token"
```

## Resources

### towerops_site

Manages a TowerOps site. Sites represent physical locations that contain devices.

```hcl
resource "towerops_site" "example" {
  name           = "Main Office"
  location       = "New York, NY"
  snmp_community = "public"
}
```

#### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Site name (2-200 characters) |
| `location` | string | No | Physical location or address |
| `snmp_community` | string | No | Default SNMP community string for devices |

#### Attributes

| Name | Description |
|------|-------------|
| `id` | Unique identifier (UUID) |
| `inserted_at` | Creation timestamp |

#### Import

```bash
terraform import towerops_site.example 550e8400-e29b-41d4-a716-446655440000
```

### towerops_device

Manages a TowerOps device. Devices represent network equipment at a site.

```hcl
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

#### Arguments

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `site_id` | string | Yes | - | ID of the parent site (forces replacement if changed) |
| `name` | string | Yes | - | Device name |
| `ip_address` | string | Yes | - | Device IP address |
| `description` | string | No | - | Device description |
| `monitoring_enabled` | bool | No | `true` | Enable monitoring |
| `snmp_enabled` | bool | No | `true` | Enable SNMP polling |
| `snmp_version` | string | No | `"2c"` | SNMP version (1, 2c, or 3) |
| `snmp_port` | number | No | `161` | SNMP port |

#### Attributes

| Name | Description |
|------|-------------|
| `id` | Unique identifier (UUID) |
| `inserted_at` | Creation timestamp |

#### Import

```bash
terraform import towerops_device.router 7c9e6679-7425-40de-944b-e07fc1f90ae7
```

## Example

```hcl
terraform {
  required_providers {
    towerops = {
      source  = "towerops/towerops"
      version = "~> 0.1"
    }
  }
}

variable "towerops_api_token" {
  type      = string
  sensitive = true
}

provider "towerops" {
  token = var.towerops_api_token
}

resource "towerops_site" "datacenter" {
  name           = "Primary Datacenter"
  location       = "Chicago, IL"
  snmp_community = "monitoring"
}

resource "towerops_device" "core_switch" {
  site_id     = towerops_site.datacenter.id
  name        = "Core Switch"
  ip_address  = "10.0.0.1"
  description = "Main distribution switch"

  snmp_enabled = true
  snmp_version = "2c"
}

resource "towerops_device" "edge_router" {
  site_id    = towerops_site.datacenter.id
  name       = "Edge Router"
  ip_address = "10.0.0.2"

  snmp_enabled = true
  snmp_version = "3"
}

output "site_id" {
  value = towerops_site.datacenter.id
}
```

## License

MPL-2.0
