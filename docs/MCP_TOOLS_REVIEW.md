# MCP Tools Review: Outcome-Based, Agent-Friendly Design

## Goals

- **Outcome-based tooling**: Tools express user intent and outcomes, not low-level API steps.
- **Agent-friendly**: Flat parameters; minimal tool count; clear descriptions and errors.
- **Common VM/Container tasks**: Create, modify, lifecycle, stats, and events.
- **Delete**: Supported but **require explicit confirmation** to avoid accidental loss.

---

## Current State Summary

| Aspect | Current | Notes |
|--------|---------|--------|
| **Tool count** | ~107 total (~50 default) | High; many similar tools (e.g. start/stop/shutdown/reboot per resource type). |
| **Parameters** | Mostly flat | **Exceptions**: `update_vm_config`, `update_container_config`, `update_storage`, `update_node_config`, `create_storage` use a nested `config` object. |
| **Delete** | No confirmation | All delete_* tools execute immediately. |
| **Stats** | Present | `get_node_stats`, `get_vm_stats`, `get_container_stats`. |
| **Events / analysis** | Tasks only | `get_node_tasks`, `get_cluster_tasks`, `get_task_status`, `get_task_log`. No cluster log/audit events. |

---

## 1. Outcome-Based Tooling

### 1.1 What “outcome-based” means

- **Good**: “Create a VM with 2 cores and 4GB RAM” → one tool, flat params.
- **Bad**: “Create VM, then update config, then add disk” as multiple low-level steps when the outcome is “one VM with disk”.

Current tools are already outcome-oriented for many cases (e.g. `create_vm`, `clone_vm`, `migrate_vm`). Improvements:

- **Single “create” entry points** with optional flat params instead of separate basic vs advanced (see consolidation below).
- **Single “control” entry point** for lifecycle (start/stop/shutdown/reboot) so the agent picks one action instead of four tools per resource type.
- **Unified “list/describe”** where it makes sense (e.g. one way to list “guests” with optional type filter).

### 1.2 Consolidation ideas (fewer tools, same outcomes)

| Outcome | Current tools | Proposed consolidation |
|--------|----------------|-------------------------|
| List cluster view | `get_nodes`, `get_cluster_resources`, `get_cluster_status` | **`get_cluster_overview`** – one call returns nodes, resources, and status (or keep 2: overview + status if payload size is a concern). |
| List guests | `get_vms`, `get_containers` | **`list_guests`** – params: `node_name`, `type` = `vm` \| `lxc` \| `all`. |
| Guest lifecycle | `start_vm`, `stop_vm`, `shutdown_vm`, `reboot_vm` (+ container equivalents) | **`control_guest`** – params: `node_name`, `guest_type` (vm \| lxc), `vmid` (id), `action` = start \| stop \| shutdown \| reboot. One tool for VMs and LXC. |
| Create VM | `create_vm`, `create_vm_advanced` | **`create_vm`** – single tool; all current advanced options as optional flat params (memory, cores, sockets, name, ide2, sata0, net0, etc.). |
| Create container | `create_container`, `create_container_advanced` | **`create_container`** – same idea; one tool, flat optional params. |
| Update VM/container | `update_vm_config`, `update_container_config` | **`update_guest_config`** – flat optional params (memory, cores, name, template, net0, etc.) instead of a single `config` object (see Flat params below). |
| Snapshots | create/list/delete/restore × VM + container | **`create_snapshot`**, **`list_snapshots`**, **`delete_snapshot`**, **`restore_snapshot`** – each takes `guest_type` (vm \| lxc) and common ids (node, vmid/snap_name). |
| Get guest info | `get_vm_status`, `get_vm_config`, `get_vm_stats` (and container equivalents) | Optional: **`get_guest_info`** with `detail` = status \| config \| stats \| all to reduce tool count; or keep separate getters if agents prefer explicit names. |

This reduces tool count while preserving outcomes: create, modify, lifecycle, list, snapshot, and stats.

---

## 2. Flat Parameters (agent-friendly)

### 2.1 Current non-flat parameters

- **`update_vm_config`**: `config` (object) – agent must build JSON (e.g. `{"template": 1}`, `{"memory": 2048}`).
- **`update_container_config`**: same.
- **`update_storage`** / **`update_node_config`**: `config` (object).
- **`create_storage`**: `config` (object) for type-specific options.

Agents and users are better served by **flat, optional** fields for common cases.

### 2.2 Recommended approach

- **VM config updates**: Add flat optional params, e.g.  
  `memory`, `cores`, `sockets`, `name`, `template`, `net0`, `sata0`, `ide2`, etc.  
  Handler builds the config map from present params. Keep a single optional `config` (object) only for rare/arbitrary keys.
- **Container config updates**: Same – `memory`, `cores`, `hostname`, `net0`, `rootfs`, etc. as flat optionals.
- **Storage/Node**: For `update_storage` / `update_node_config`, add flat params for the most common options; keep `config` for the rest.

This keeps the schema simple for the model (no nested structure for common paths) and avoids JSON-in-JSON mistakes.

---

## 3. Stats and Events for Analysis

### 3.1 Stats (already present)

- `get_node_stats` – node performance.
- `get_vm_stats` – VM CPU/mem/disk (from status/current).
- `get_container_stats` – container stats.

These support “how is my cluster/VM/container doing?” outcomes. Optional: **`get_cluster_stats`** that aggregates node (and optionally guest) stats for a single “cluster health” view.

### 3.2 Events / audit trail

- **Current**: Tasks only – `get_cluster_tasks`, `get_node_tasks`, `get_task_status`, `get_task_log`. Good for “what’s running?” and “why did X fail?”.
- **Gap**: No cluster-level log or event stream for audit or “what changed?” analysis.

**Recommendation**: Add a single outcome-oriented tool, e.g. **`get_cluster_events`** (or **`get_cluster_log`**), that:

- Calls Proxmox cluster log / journal API (e.g. `cluster/log` or equivalent), if available.
- Returns recent cluster events (node join/leave, config changes, etc.) with optional filters (time range, severity).
- Enables “show me recent cluster activity” and “analyze changes” without multiple ad-hoc calls.

Implementation note: Proxmox supports cluster log retrieval (see forum "Cluster Logs Retrieval from Proxmox API"); log entries include a `pri` (priority) field 0–7 (panic … debug). A **`get_cluster_log`** tool can wrap this for recent cluster events and audit-style analysis.

---

## 4. Delete: Require Confirmation

### 4.1 Requirement

All delete operations must **require explicit confirmation** so that agents (and users) cannot delete resources by mistake.

### 4.2 Design options

**Option A – Confirmation token (recommended)**

- Every delete tool gets a required parameter, e.g. **`confirm`** (string).
- To succeed, the caller must pass the exact resource identifier (or a dedicated confirmation string) to prove intent.
- Examples:
  - `delete_vm`: require `confirm` = `"vm-<vmid>"` or `"<node_name>/<vmid>"`.
  - `delete_container`: require `confirm` = `"lxc-<container_id>"` or `"<node_name>/<container_id>"`.
  - `delete_backup`: require `confirm` = `"<storage>/<backup_id>"`.
- Server returns a clear error if `confirm` is missing or wrong: e.g. “Delete requires confirm='node_name/vmid' to proceed.”

**Option B – Two-step (preview then confirm)**

- Step 1: **`delete_vm_preview`** (or generic **`delete_preview`**) with node, vmid → returns summary (name, disks, etc.) and a short-lived **`confirmation_token`**.
- Step 2: **`delete_vm`** with the same ids **and** `confirmation_token` from the preview. Server only executes if token is valid and matches.
- Better for “show user what will be deleted, then confirm” flows; slightly more tools and round-trips.

**Option C – Boolean flag with strong naming**

- Require **`confirm_delete`** = `true` (and optionally **`confirm_resource_id`** = `"node/vmid"`).  
- Simpler than tokens but easier to misuse (agent might always pass `true`). Prefer A or B for safety.

### 4.3 Recommended implementation (Option A)

- Add **`confirm`** (string, required) to every delete tool.
- Document: “Must set confirm to the value '<node>/<vmid>' (e.g. 'pve1/100') to confirm deletion.”
- Server logic:
  - If `confirm` != expected value → return error, do not delete.
  - If `confirm` matches → proceed with delete.

Delete tools to update:

- `delete_vm`, `delete_container`, `delete_vm_snapshot`, `delete_container_snapshot`
- `delete_user`, `delete_group`, `delete_role`, `delete_api_token`
- `delete_backup`, `delete_storage`, `delete_pool`, `delete_firewall_rule`

---

## 5. Minimal Tool Set for Best Outcomes (summary)

A reduced, outcome-oriented set could look like this (without removing any current capability):

### 5.1 Cluster & discovery

- **`get_cluster_overview`** – nodes + resources (and optionally status) in one call.
- **`get_cluster_status`** – keep if separate from overview.
- **`list_guests`** – VMs and/or LXC with `node_name`, `type` = vm \| lxc \| all.

### 5.2 Guest lifecycle (VMs and containers)

- **`control_guest`** – start \| stop \| shutdown \| reboot; params: node, guest_type, vmid, action.
- **`create_vm`** – one tool, all current options as flat optional params.
- **`create_container`** – one tool, flat optional params.
- **`update_guest_config`** – VM or LXC; flat optional params (no bare `config` object for common cases).
- **`clone_guest`** – clone VM or container; params: guest_type, node, source_id, new_id, new_name, full.
- **`get_guest_info`** (optional) – status/config/stats with `detail` param; or keep separate get_vm_status, get_vm_config, get_vm_stats and container equivalents.

### 5.3 Snapshots

- **`create_snapshot`**, **`list_snapshots`**, **`restore_snapshot`** – with `guest_type` (vm \| lxc).
- **`delete_snapshot`** – same; **requires `confirm`** (e.g. `node/vmid/snapname`).

### 5.4 Stats and events

- **`get_node_stats`**, **`get_vm_stats`**, **`get_container_stats`** (or **`get_guest_stats`** with guest_type).
- **`get_cluster_tasks`**, **`get_task_status`**, **`get_task_log`**.
- **`get_cluster_events`** (new) – cluster log/events for analysis.

### 5.5 Delete (all require `confirm`)

- **`delete_vm`**, **`delete_container`** – require `confirm` = `"<node>/<vmid>"`.
- **`delete_snapshot`** – require `confirm` = `"<node>/<vmid>/<snap_name>"` (or equivalent for LXC).
- **`delete_backup`**, **`delete_storage`**, **`delete_pool`**, **`delete_firewall_rule`** – require `confirm` with resource identifier.
- User/ACL: **`delete_user`**, **`delete_group`**, **`delete_role`**, **`delete_api_token`** – require `confirm` = userid/groupid/roleid/tokenid.

### 5.6 Rest unchanged

- Storage, backups, restore, pools, HA, firewall, node config, etc. can stay as today but:
  - Prefer flat params where possible (e.g. update_vm_config / update_container_config).
  - Add **`confirm`** to all delete_* tools.

---

## 6. Implementation Priority

1. **Delete confirmation** – Add required `confirm` to all delete tools (quick, high safety impact).
2. **Flat params** – Refactor `update_vm_config` and `update_container_config` to accept flat optional params; keep optional `config` for edge cases.
3. **Events** – Add **`get_cluster_events`** (or **`get_cluster_log`**) once the correct Proxmox API is confirmed.
4. **Consolidation** – Introduce unified tools (e.g. `list_guests`, `control_guest`, single `create_vm`/`create_container`) alongside existing ones; deprecate duplicates later, or keep both during transition.

This order maximizes safety and agent usability with minimal breaking change, then improves analysis and tool count.
