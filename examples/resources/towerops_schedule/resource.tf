resource "towerops_schedule" "primary" {
  name        = "Primary On-Call"
  timezone    = "America/Chicago"
  description = "Main engineering on-call rotation"
}
