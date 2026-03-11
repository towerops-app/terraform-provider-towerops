---
page_title: "towerops_escalation_policy Resource - TowerOps"
description: |-
  Manages a TowerOps escalation policy.
---

# towerops_escalation_policy (Resource)

Manages a TowerOps escalation policy. Escalation policies define how alerts are routed through a series of notification rules when they are not acknowledged.

## Example Usage

### Basic Escalation Policy

```terraform
resource "towerops_escalation_policy" "default" {
  name = "Default Escalation"
}
```

### Escalation Policy with Custom Repeat Count

```terraform
resource "towerops_escalation_policy" "critical" {
  name         = "Critical Alerts"
  description  = "Escalation for P1 incidents"
  repeat_count = 5
}
```

## Schema

### Required

- `name` (String) - The name of the escalation policy.

### Optional

- `description` (String) - A description of the escalation policy.
- `repeat_count` (Number) - Number of times to repeat the escalation cycle. Defaults to `3`.

### Read-Only

- `id` (String) - The unique identifier of the escalation policy.
- `inserted_at` (String) - The timestamp when the escalation policy was created.

## Import

Escalation policies can be imported using their UUID:

```shell
terraform import towerops_escalation_policy.example 550e8400-e29b-41d4-a716-446655440000
```
