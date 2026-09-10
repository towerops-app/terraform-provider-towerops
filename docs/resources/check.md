---
page_title: "towerops_check Resource - TowerOps"
description: |-
  Manages a TowerOps service check.
---

# towerops_check (Resource)

Manages a TowerOps service check. Service checks monitor the availability and responsiveness of network services. Supported check types: HTTP, TCP, DNS, and ping.

Configuration fields are flattened into top-level attributes rather than nested inside a config block. Only the fields relevant to the chosen `check_type` are used.

## Example Usage

### HTTP Health Check

```terraform
resource "towerops_check" "web_health" {
  name            = "Web Health Check"
  check_type      = "http"
  url             = "https://example.com/health"
  expected_status = 200
  content_match   = "\"status\":\"ok\""
}
```

### HTTP Check with Custom Settings

```terraform
resource "towerops_check" "api_check" {
  name             = "API Endpoint"
  check_type       = "http"
  device_id        = towerops_device.web_server.id
  url              = "https://api.example.com/v1/status"
  method           = "POST"
  expected_status  = 200
  verify_ssl       = true
  follow_redirects = false
  interval_seconds = 30
  timeout_ms       = 10000
}
```

### TCP Port Check

```terraform
resource "towerops_check" "radius_port" {
  name       = "FreeRADIUS Auth Port"
  check_type = "tcp"
  device_id  = towerops_device.radius.id
  host       = "10.0.0.5"
  port       = 1812
}
```

### TCP Check with Send/Expect

```terraform
resource "towerops_check" "smtp_banner" {
  name          = "SMTP Banner Check"
  check_type    = "tcp"
  host          = "mail.example.com"
  port          = 25
  expect_string = "220"
}
```

### DNS Resolution Check

```terraform
resource "towerops_check" "dns_resolution" {
  name            = "DNS Resolution"
  check_type      = "dns"
  device_id       = towerops_device.dns_server.id
  hostname        = "google.com"
  dns_server      = "10.0.0.1"
  record_type     = "A"
  expected_result = "142.250.80.46"
}
```

### Ping Check

```terraform
resource "towerops_check" "gateway_ping" {
  name       = "Gateway Reachability"
  check_type = "ping"
  device_id  = towerops_device.gateway.id
  host       = "10.0.0.1"
}
```

### Ping Check with Thresholds

```terraform
resource "towerops_check" "wan_ping" {
  name             = "WAN Link Ping"
  check_type       = "ping"
  host             = "8.8.8.8"
  ping_count       = 5
  interval_seconds = 120
  timeout_ms       = 10000

  loss_warning_percent = 2.0
  latency_warning_ms   = 100
  latency_critical_ms  = 250
}
```

### HTTP Check with Headers and a Body

```terraform
resource "towerops_check" "ingest_probe" {
  name            = "Ingest Probe"
  check_type      = "http"
  url             = "https://api.example.com/v1/probe"
  method          = "POST"
  expected_status = 202
  request_body    = jsonencode({ probe = true })

  request_headers = {
    "Content-Type" = "application/json"
    "X-Probe"      = "towerops"
  }
}
```

## Schema

### Required

- `name` (String) - The name of the check.
- `check_type` (String) - The type of check. The REST API accepts only `http`, `tcp`, `dns`, and `ping`; any other value is rejected with a 400 `bad_request`. Changing this forces a new resource.

### Optional

- `description` (String) - A description of the check.
- `enabled` (Boolean) - Whether the check is enabled. Default: `true`.
- `alerting` (Boolean) - Whether the check raises alerts. Update-only: the create endpoint ignores this field and arms every new check, so `alerting = false` on a check that does not exist yet is applied on the next apply, once the check can be updated. Default: `true`.
- `device_id` (String) - The ID of the device this check is associated with.
- `agent_token_id` (String) - The ID of the agent token that executes this check.
- `interval_seconds` (Number) - How often the check runs, in seconds. Default: `60`.
- `timeout_ms` (Number) - Timeout for check execution, in milliseconds. Default: `5000`.
- `retry_interval_seconds` (Number) - Interval between retries after a failure, in seconds. Default: `30`.
- `max_check_attempts` (Number) - Number of consecutive failures before transitioning to hard state. Default: `3`.

#### HTTP Check Fields (used when `check_type = "http"`)

- `url` (String) - The URL to check. Required for HTTP checks.
- `method` (String) - HTTP method to use. Default: `"GET"`.
- `expected_status` (Number) - Expected HTTP status code. Default: `200`.
- `verify_ssl` (Boolean) - Whether to verify SSL certificates. Default: `true`.
- `follow_redirects` (Boolean) - Whether to follow HTTP redirects. Default: `true`.
- `content_match` (String) - Regex pattern to match against the response body.
- `request_headers` (Map of String) - Additional request headers sent with the check.
- `request_body` (String) - Request body sent with the check.

#### TCP Check Fields (used when `check_type = "tcp"`)

- `host` (String) - The host to connect to. Required for TCP checks.
- `port` (Number) - The TCP port to check. Required for TCP checks.
- `send_string` (String) - Data to send after establishing the TCP connection.
- `expect_string` (String) - Expected response string.

#### DNS Check Fields (used when `check_type = "dns"`)

- `hostname` (String) - The hostname to resolve. Required for DNS checks.
- `record_type` (String) - DNS record type (`A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `PTR`). Default: `"A"`.
- `dns_server` (String) - The DNS server to query, sent as the `server` config key. Defaults to the resolver of the executing host.
- `expected_result` (String) - Expected DNS resolution result.

#### Ping Check Fields (used when `check_type = "ping"`)

- `host` (String) - The host to ping. Required for ping checks.
- `ping_count` (Number) - Number of ping packets to send. Default: `3`. The executor clamps the value to 1 through 10.
- `loss_warning_percent` (Number) - Packet loss percentage above which the check reports WARNING, 0 through 100. The executor default is `0.0`, so any packet loss warns.
- `latency_warning_ms` (Number) - Average round trip time in milliseconds above which the check reports WARNING. Must be positive and below `latency_critical_ms`. No threshold is applied when unset.
- `latency_critical_ms` (Number) - Average round trip time in milliseconds above which the check reports CRITICAL. Must be positive. No threshold is applied when unset.

~> Packet loss is only graded when the check runs through the ping executor. A
result ingested from an agent carries status and response time but not loss, so
`loss_warning_percent` cannot be applied to it and grading falls back to the
latency thresholds alone.

### Read-Only

- `id` (String) - The unique identifier of the check.
- `current_state` (Number) - Current check state (0=OK, 1=WARNING, 2=CRITICAL, 3=UNKNOWN).
- `current_state_type` (String) - Current state type (`soft` or `hard`).
- `last_check_at` (String) - Timestamp of the last check execution.
- `inserted_at` (String) - The timestamp when the check was created.

## Import

Checks can be imported using their UUID:

```shell
terraform import towerops_check.web_health 7c9e6679-7425-40de-944b-e07fc1f90ae7
```
