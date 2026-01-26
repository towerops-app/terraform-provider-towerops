---
page_title: "TowerOps Provider"
description: |-
  The TowerOps provider allows you to manage TowerOps resources such as sites and devices.
---

# TowerOps Provider

The TowerOps provider allows you to manage [TowerOps](https://towerops.io) resources via Terraform.

## Authentication

The provider requires an API token for authentication. Generate a token from the TowerOps web application under Settings → API Tokens. The token determines which organization's resources are accessible.

## Example Usage

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

## Schema

### Required

- `token` (String, Sensitive) - The API token for authenticating with TowerOps.
