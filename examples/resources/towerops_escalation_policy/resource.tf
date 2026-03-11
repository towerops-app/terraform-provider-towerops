resource "towerops_escalation_policy" "critical" {
  name         = "Critical Alerts"
  description  = "Escalation for P1 incidents"
  repeat_count = 5
}
