# HTTP health check
resource "towerops_check" "web_health" {
  name            = "Web Health Check"
  check_type      = "http"
  url             = "https://example.com/health"
  expected_status = 200
  content_match   = "\"status\":\"ok\""
}

# TCP port check
resource "towerops_check" "radius_port" {
  name       = "FreeRADIUS Auth Port"
  check_type = "tcp"
  host       = "10.0.0.5"
  port       = 1812
}

# DNS resolution check
resource "towerops_check" "dns_resolution" {
  name        = "DNS Resolution"
  check_type  = "dns"
  hostname    = "google.com"
  dns_server  = "10.0.0.1"
  record_type = "A"
}

# Ping check
resource "towerops_check" "gateway_ping" {
  name       = "Gateway Reachability"
  check_type = "ping"
  host       = "10.0.0.1"
}
