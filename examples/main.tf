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

# Manage organization settings
resource "towerops_organization" "main" {
  name      = "My Organization"
  use_sites = true
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

# Create an on-call schedule
resource "towerops_schedule" "primary" {
  name        = "Primary On-Call"
  timezone    = "America/Chicago"
  description = "Main engineering on-call rotation"
}

# Create an escalation policy
resource "towerops_escalation_policy" "default" {
  name         = "Default Escalation"
  description  = "Standard escalation for all alerts"
  repeat_count = 3
}

# Create an agent token
resource "towerops_agent" "remote" {
  name = "Remote Site Poller"
}

# Create an integration
resource "towerops_integration" "pagerduty" {
  provider_type = "pagerduty"
  enabled  = true
}

# Create a maintenance window
resource "towerops_maintenance_window" "network_upgrade" {
  name      = "Network Upgrade"
  reason    = "Upgrading core switches"
  starts_at = "2024-03-15T02:00:00Z"
  ends_at   = "2024-03-15T06:00:00Z"
}

# Output the site ID
output "site_id" {
  value = towerops_site.main_office.id
}

output "router_id" {
  value = towerops_device.core_router.id
}

output "schedule_id" {
  value = towerops_schedule.primary.id
}

output "escalation_policy_id" {
  value = towerops_escalation_policy.default.id
}

output "agent_token" {
  value     = towerops_agent.remote.token
  sensitive = true
}
