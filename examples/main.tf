terraform {
  required_providers {
    towerops = {
      source = "towerops/towerops"
    }
  }
}

variable "towerops_api_token" {
  description = "TowerOps API token"
  type        = string
  sensitive   = true
}

provider "towerops" {
  token = var.towerops_api_token
}

# Create a site
resource "towerops_site" "main_office" {
  name           = "Main Office"
  location       = "New York, NY"
  snmp_community = "public"
}

# Create devices at the site
resource "towerops_device" "core_router" {
  site_id    = towerops_site.main_office.id
  name       = "Core Router"
  ip_address = "192.168.1.1"

  monitoring_enabled = true
  snmp_enabled       = true
  snmp_version       = "2c"
  snmp_port          = 161
}

resource "towerops_device" "access_switch" {
  site_id    = towerops_site.main_office.id
  name       = "Access Switch"
  ip_address = "192.168.1.2"

  monitoring_enabled = true
  snmp_enabled       = true
}

# Output the site ID
output "site_id" {
  value = towerops_site.main_office.id
}

output "router_id" {
  value = towerops_device.core_router.id
}
