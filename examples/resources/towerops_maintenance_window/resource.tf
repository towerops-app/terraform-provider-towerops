resource "towerops_maintenance_window" "network_upgrade" {
  name      = "Network Upgrade"
  reason    = "Upgrading core switches to new firmware"
  starts_at = "2024-03-15T02:00:00Z"
  ends_at   = "2024-03-15T06:00:00Z"
}
