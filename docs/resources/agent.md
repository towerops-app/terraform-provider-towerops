---
page_title: "towerops_agent Resource - TowerOps"
description: |-
  Manages a TowerOps agent token.
---

# towerops_agent (Resource)

Manages a TowerOps agent token. Agents are deployed on customer networks to poll devices via SNMP, ping, and SSH. The agent token is returned only on creation and cannot be retrieved again.

~> **Note:** The `token` attribute is only available after creation. If the state is lost, the agent must be deleted and recreated to obtain a new token.

~> **Note:** The TowerOps API has no update action for agents. Changing `name` replaces the agent, which issues a new token, and every other attribute is read-only.

## Example Usage

### Basic Agent

```terraform
resource "towerops_agent" "office" {
  name = "Office Poller"
}
```

### Using the Token

```terraform
resource "towerops_agent" "remote" {
  name = "Remote Site Poller"
}

output "agent_token" {
  value     = towerops_agent.remote.token
  sensitive = true
}
```

## Schema

### Required

- `name` (String) - The name of the agent. Changing this forces a new resource to be created.

### Read-Only

- `id` (String) - The unique identifier of the agent.
- `token` (String, Sensitive) - The bearer token for this agent. Only available after creation and cannot be retrieved again.
- `last_ip` (String) - The source IP address the agent last checked in from. Null until the agent checks in.
- `metadata` (Map of String) - Metadata the agent reported on its last check-in, such as its version and host details. The API sends an arbitrary JSON object, so values that are not strings (numbers, booleans, nested objects, arrays) are stored as their JSON encoding. Null when the agent has reported nothing.
- `device_count` (Number) - The number of devices currently assigned to this agent. `0` for a newly created agent, since the create response does not report it.
- `inserted_at` (String) - The timestamp when the agent was created.

## Import

Agents can be imported using their UUID. Note that the token will not be available after import.

```shell
terraform import towerops_agent.example 550e8400-e29b-41d4-a716-446655440000
```
