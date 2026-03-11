---
page_title: "towerops_agent Resource - TowerOps"
description: |-
  Manages a TowerOps agent token.
---

# towerops_agent (Resource)

Manages a TowerOps agent token. Agents are deployed on customer networks to poll devices via SNMP, ping, and SSH. The agent token is returned only on creation and cannot be retrieved again.

~> **Note:** The `token` attribute is only available after creation. If the state is lost, the agent must be deleted and recreated to obtain a new token.

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
- `inserted_at` (String) - The timestamp when the agent was created.

## Import

Agents can be imported using their UUID. Note that the token will not be available after import.

```shell
terraform import towerops_agent.example 550e8400-e29b-41d4-a716-446655440000
```
