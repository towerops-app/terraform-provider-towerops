# TowerOps Terraform Provider

Terraform provider for managing [TowerOps](https://towerops.net) resources.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.26 (to build the provider)

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

The provider requires an API token. Generate one in the TowerOps web
application under Settings, API Tokens. The token determines which
organization's resources are reachable.

```hcl
provider "towerops" {
  token = var.towerops_api_token
}
```

To point at a self-hosted or development instance:

```hcl
provider "towerops" {
  token   = var.towerops_api_token
  api_url = "https://custom.example.com"
}
```

`token` is a required provider argument. The provider does not read it from the
environment; wire an environment variable in through a Terraform variable if
you want one:

```hcl
variable "towerops_api_token" {
  type      = string
  sensitive = true
}
```

```bash
export TF_VAR_towerops_api_token="your-api-token"
```

### Token scopes

TowerOps API tokens are scoped, and an endpoint called with a token that lacks
its scope answers HTTP 403 with the code `insufficient_scope`. Grant the token
every scope your configuration needs:

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

## Resources

| Resource | Manages | Reference |
| --- | --- | --- |
| `towerops_organization` | Settings for the organization the token belongs to | [docs](docs/index.md) |
| `towerops_site` | A physical location that groups devices | [docs](docs/resources/site.md) |
| `towerops_device` | Network equipment, at a site or directly in the organization | [docs](docs/resources/device.md) |
| `towerops_check` | HTTP, TCP, DNS, and ping service checks | [docs](docs/resources/check.md) |
| `towerops_agent` | An agent token for a remote poller | [docs](docs/resources/agent.md) |
| `towerops_schedule` | An on-call schedule | [docs](docs/resources/schedule.md) |
| `towerops_escalation_policy` | An alert escalation policy | [docs](docs/resources/escalation_policy.md) |
| `towerops_maintenance_window` | A one-off or recurring maintenance window | [docs](docs/resources/maintenance_window.md) |
| `towerops_integration` | A third-party integration such as PagerDuty or NetBox | [docs](docs/resources/integration.md) |
| `towerops_coverage` | An RF coverage prediction for one antenna | [docs](docs/resources/coverage.md) |
| `towerops_webhook_endpoint` | An outbound webhook subscription | [docs](docs/resources/webhook_endpoint.md) |

Every resource supports `terraform import` with the resource UUID.

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

resource "towerops_check" "gateway_ping" {
  name       = "Gateway Reachability"
  check_type = "ping"
  host       = "10.0.0.1"
  device_id  = towerops_device.core_switch.id
}

output "site_id" {
  value = towerops_site.datacenter.id
}
```

See [`examples/main.tf`](examples/main.tf) for a configuration that exercises
every resource.

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

The acceptance suite drives real Terraform against in-process mock API servers,
so it needs a `terraform` binary but no credentials and no network access:

```bash
TF_ACC=1 go test -count=1 ./internal/provider/...
```

Set `TF_ACC_TERRAFORM_PATH=/path/to/terraform` to use a terraform binary you
already have instead of letting the test harness download one.

## License

MPL-2.0
