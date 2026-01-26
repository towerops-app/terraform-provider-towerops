# TowerOps Terraform Provider Design

## Overview

A Terraform provider for managing TowerOps resources (Sites and Devices) via the existing REST API.

## Scope

- **Resources**: Sites, Devices
- **Authentication**: API token in provider config
- **Import**: Supported for existing resources
- **Endpoint**: Fixed production URL

## Provider Configuration

```hcl
terraform {
  required_providers {
    towerops = {
      source = "towerops/towerops"
    }
  }
}

provider "towerops" {
  token = var.towerops_api_token  # Required, scopes all operations to one org
}
```

The token determines which organization's resources are accessible via the existing API token authentication.

## Project Structure

```
towerops-tf-provider/
├── main.go                 # Entry point
├── go.mod / go.sum
├── internal/
│   └── provider/
│       ├── provider.go     # Provider schema + config
│       ├── client.go       # HTTP client for TowerOps API
│       ├── site_resource.go
│       ├── device_resource.go
├── examples/
│   └── main.tf             # Usage example
└── docs/                   # Generated documentation
```

Uses the Terraform Plugin Framework (modern approach).

## Resource Schemas

### towerops_site

```hcl
resource "towerops_site" "main" {
  name           = "Main Office"
  location       = "New York, NY"      # Optional
  snmp_community = "public"            # Optional
}

output "site_id" {
  value = towerops_site.main.id
}
```

**Attributes:**
- `name` (Required, string) - Site name, 2-200 chars
- `location` (Optional, string) - Physical location description
- `snmp_community` (Optional, string) - Default SNMP community for devices

**Computed:**
- `id` (string) - UUID assigned by TowerOps
- `inserted_at` (string) - Creation timestamp

### towerops_device

```hcl
resource "towerops_device" "router" {
  site_id    = towerops_site.main.id
  name       = "Core Router"
  ip_address = "192.168.1.1"

  monitoring_enabled = true    # Optional, default true
  snmp_enabled       = true    # Optional, default true
  snmp_version       = "2c"    # Optional, default "2c"
  snmp_port          = 161     # Optional, default 161
}
```

**Attributes:**
- `site_id` (Required, string) - UUID of parent site
- `name` (Required, string) - Device name
- `ip_address` (Required, string) - Device IP address
- `description` (Optional, string) - Device description
- `monitoring_enabled` (Optional, bool) - Enable monitoring, default true
- `snmp_enabled` (Optional, bool) - Enable SNMP polling, default true
- `snmp_version` (Optional, string) - SNMP version (1, 2c, 3), default "2c"
- `snmp_port` (Optional, int) - SNMP port, default 161

**Computed:**
- `id` (string) - UUID assigned by TowerOps
- `inserted_at` (string) - Creation timestamp

## HTTP Client

```go
type Client struct {
    BaseURL    string
    Token      string
    HTTPClient *http.Client
}

// Site operations
func (c *Client) CreateSite(site Site) (*Site, error)
func (c *Client) GetSite(id string) (*Site, error)
func (c *Client) UpdateSite(id string, site Site) (*Site, error)
func (c *Client) DeleteSite(id string) error

// Device operations
func (c *Client) CreateDevice(device Device) (*Device, error)
func (c *Client) GetDevice(id string) (*Device, error)
func (c *Client) UpdateDevice(id string, device Device) (*Device, error)
func (c *Client) DeleteDevice(id string) error
```

### API Mapping

| Terraform Operation | HTTP Method | Endpoint |
|---------------------|-------------|----------|
| Create | POST | `/api/v1/sites` or `/api/v1/devices` |
| Read | GET | `/api/v1/sites/:id` or `/api/v1/devices/:id` |
| Update | PATCH | `/api/v1/sites/:id` or `/api/v1/devices/:id` |
| Delete | DELETE | `/api/v1/sites/:id` or `/api/v1/devices/:id` |
| Import | GET | Same as Read |

## Import Support

```bash
# Import existing site by UUID
terraform import towerops_site.main 550e8400-e29b-41d4-a716-446655440000

# Import existing device by UUID
terraform import towerops_device.router 7c9e6679-7425-40de-944b-e07fc1f90ae7
```

Import uses the same `GetSite`/`GetDevice` client methods as Read.

## Testing Strategy

1. **Unit tests** - Mock HTTP responses, test schema validation
2. **Acceptance tests** - Run against real TowerOps instance (skipped in CI without credentials)

```go
func TestAccSiteResource(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: `resource "towerops_site" "test" { name = "Test Site" }`,
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("towerops_site.test", "name", "Test Site"),
                    resource.TestCheckResourceAttrSet("towerops_site.test", "id"),
                ),
            },
        },
    })
}
```

## Implementation Order

1. Project scaffolding (go.mod, main.go)
2. HTTP client with Site CRUD
3. Site resource implementation
4. HTTP client Device CRUD
5. Device resource implementation
6. Import support for both resources
7. Acceptance tests
8. Documentation and examples
