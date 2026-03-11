resource "towerops_integration" "pagerduty" {
  provider              = "pagerduty"
  enabled               = true
  sync_interval_minutes = 5
}
