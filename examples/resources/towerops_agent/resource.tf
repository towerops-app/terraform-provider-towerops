resource "towerops_agent" "office" {
  name = "Office Poller"
}

output "agent_token" {
  value     = towerops_agent.office.token
  sensitive = true
}
