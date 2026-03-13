resource "towerops_integration" "pagerduty" {
  provider_type         = "pagerduty"
  enabled               = true
  sync_interval_minutes = 5
}
