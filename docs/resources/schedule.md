---
page_title: "towerops_schedule Resource - TowerOps"
description: |-
  Manages a TowerOps on-call schedule.
---

# towerops_schedule (Resource)

Manages a TowerOps on-call schedule. Schedules define rotation layers that determine who is on-call at any given time.

~> **Note:** Layers, layer members and overrides are managed through separate nested endpoints (`POST /api/v1/schedules/:id/layers`, `POST /api/v1/schedules/:id/layers/:layer_id/members` and `POST /api/v1/schedules/:id/overrides`) that this provider does not expose yet. A schedule created here has no rotation, so nobody is on call, until layers and members are added in the TowerOps UI.

## Example Usage

### Basic Schedule

```terraform
resource "towerops_schedule" "primary" {
  name     = "Primary On-Call"
  timezone = "America/Chicago"
}
```

### Schedule with Description

```terraform
resource "towerops_schedule" "after_hours" {
  name        = "After-Hours Rotation"
  timezone    = "America/New_York"
  description = "Coverage for nights and weekends"
}
```

## Schema

### Required

- `name` (String) - The name of the on-call schedule.
- `timezone` (String) - The timezone for the schedule (e.g. `America/Chicago`, `UTC`).

### Optional

- `description` (String) - A description of the schedule.

### Read-Only

- `id` (String) - The unique identifier of the schedule.
- `inserted_at` (String) - The timestamp when the schedule was created.

## Import

Schedules can be imported using their UUID:

```shell
terraform import towerops_schedule.example 550e8400-e29b-41d4-a716-446655440000
```
