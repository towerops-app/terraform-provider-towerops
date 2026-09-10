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
  description    = "Head office and primary POP"
  location       = "New York, NY"
  address        = "350 5th Ave, New York, NY 10118"
  latitude       = 40.7484
  longitude      = -73.9857
  display_order  = 1
  snmp_community = "public"
  snmp_version   = "2c"
  snmp_port      = 161
}

# Create devices at the site
resource "towerops_device" "core_router" {
  site_id    = towerops_site.main_office.id
  name       = "Core Router"
  ip_address = "192.168.1.1"

  device_role            = "router"
  monitoring_enabled     = true
  check_interval_seconds = 300
  snmp_enabled           = true
  snmp_version           = "2c"
  snmp_port              = 161
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

# Create an integration. Credentials are configured in the TowerOps UI: the
# REST API only accepts the provider, its enabled flag, and the sync interval.
resource "towerops_integration" "pagerduty" {
  provider_type = "pagerduty"
  enabled       = true
}

# Create a maintenance window
resource "towerops_maintenance_window" "network_upgrade" {
  name      = "Network Upgrade"
  reason    = "Upgrading core switches"
  starts_at = "2026-03-15T02:00:00Z"
  ends_at   = "2026-03-15T06:00:00Z"
}

# A recurring window, expressed as an RRULE
resource "towerops_maintenance_window" "weekly_patching" {
  name            = "Weekly Patching"
  reason          = "Routine firmware patching"
  starts_at       = "2026-03-01T03:00:00Z"
  ends_at         = "2026-03-01T05:00:00Z"
  recurring       = true
  recurrence_rule = "FREQ=WEEKLY;BYDAY=SU"
  suppress_alerts = true
  site_id         = towerops_site.main_office.id
}

# Service checks
resource "towerops_check" "web_health" {
  name            = "Web Health Check"
  check_type      = "http"
  url             = "https://example.com/health"
  expected_status = 200
  content_match   = "\"status\":\"ok\""
}

resource "towerops_check" "dns_resolution" {
  name        = "DNS Resolution"
  check_type  = "dns"
  hostname    = "google.com"
  dns_server  = "10.0.0.1"
  record_type = "A"
}

resource "towerops_check" "gateway_ping" {
  name       = "Gateway Reachability"
  check_type = "ping"
  host       = "10.0.0.1"
}

# RF coverage prediction for one sector. Creating it enqueues a compute job,
# so `status` reads "queued" immediately after apply and reaches "ready" once
# the worker finishes. All inputs are SI; the API converts imperial inputs
# before storing them, which would make every plan dirty.
resource "towerops_coverage" "north_sector" {
  name          = "North sector 5 GHz"
  site_id       = towerops_site.main_office.id
  antenna_slug  = "rf-elements-tp-sh-30"
  frequency_mhz = 5800
  tx_power_dbm  = 22.0
  height_agl_m  = 30.0
  azimuth_deg   = 0
  downtilt_deg  = 3.0
  radius_m      = 6437
}

# Outbound webhook subscription. Requires a token with the webhooks:manage
# scope. Omitting `secret` makes the server generate one, returned once.
resource "towerops_webhook_endpoint" "alerts" {
  name = "Alert bridge"
  url  = "https://hooks.example.com/towerops/alerts"
  events = [
    "alert.triggered",
    "alert.acknowledged",
    "alert.resolved",
  ]
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

output "coverage_status" {
  value = towerops_coverage.north_sector.status
}

output "webhook_secret" {
  value     = towerops_webhook_endpoint.alerts.secret
  sensitive = true
}
