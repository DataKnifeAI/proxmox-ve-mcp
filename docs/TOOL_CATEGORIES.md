# Tool Categories and Lazy Loading

This MCP server implements **lazy loading** to reduce LLM confusion by categorizing tools into **default** (common) and **advanced** (specialized) categories.

## How It Works

Tools are defined but only registered with the MCP server based on environment variable configuration:

- **Default tools** (~40-50): Common operations enabled by default
- **Advanced tools** (~60-70): Specialized operations that require explicit enablement

## Environment Variables

### `MCP_ENABLE_ADVANCED_TOOLS`
Set to `true` to enable all advanced tools.

```bash
export MCP_ENABLE_ADVANCED_TOOLS=true
```

### `MCP_TOOLS_MODE`
Alternative way to control tool registration:
- `default` - Only default tools (same as not setting `MCP_ENABLE_ADVANCED_TOOLS`)
- `all` - All tools enabled (same as `MCP_ENABLE_ADVANCED_TOOLS=true`)

```bash
export MCP_TOOLS_MODE=all
```

## Default Tools (Always Enabled)

These are the most commonly used operations:

### Cluster & Node Management
- `get_nodes` - List all nodes
- `get_node_status` - Get node status
- `get_cluster_resources` - List all resources
- `get_cluster_status` - Cluster status

### Storage Management
- `get_storage` - List storage devices
- `get_node_storage` - Node storage info
- `get_storage_info` - Storage details
- `get_storage_content` - List storage contents

### Virtual Machine Management
- `get_vms` - List VMs
- `get_vm_status` - VM status
- `get_vm_config` - VM configuration
- `start_vm` - Start VM
- `stop_vm` - Stop VM
- `shutdown_vm` - Graceful shutdown
- `reboot_vm` - Reboot VM
- `create_vm` - Create VM (basic)
- `create_vm_advanced` - Create VM (advanced configuration)
- `clone_vm` - Clone VM
- `update_vm_config` - Update VM configuration
- `delete_vm` - Delete VM
- `suspend_vm` - Suspend VM
- `resume_vm` - Resume VM
- `get_vm_console` - Get console access
- `create_vm_snapshot` - Create VM snapshot
- `list_vm_snapshots` - List VM snapshots
- `delete_vm_snapshot` - Delete VM snapshot
- `restore_vm_snapshot` - Restore VM snapshot
- `get_vm_firewall_rules` - Get VM firewall rules
- `migrate_vm` - Migrate VM
- `get_vm_stats` - VM statistics

### User Management (Read-only)
- `list_users` - List users
- `get_user` - Get user details
- `list_groups` - List groups
- `list_roles` - List roles
- `list_acl` - List ACLs

### Task Management
- `get_node_tasks` - Node tasks
- `get_cluster_tasks` - Cluster tasks
- `get_task_status` - Task status
- `get_task_log` - Task logs

## Advanced Tools (Require Enablement)

These are specialized operations that are less commonly used:

### Cluster Operations
- `get_cluster_config` - Cluster configuration
- `get_cluster_nodes_status` - All nodes status
- `add_node_to_cluster` - Add node to cluster
- `remove_node_from_cluster` - Remove node from cluster

### Node Management (Destructive)
- `reboot_node` - Reboot a node
- `shutdown_node` - Shutdown a node
- `update_node_config` - Modify node settings
- `get_node_disks` - Physical disks
- `get_node_cert` - SSL certificates
- `get_node_logs` - System logs
- `get_node_apt_updates` - Package updates
- `apply_node_updates` - Install updates
- `get_node_network` - Network config
- `get_node_dns` - DNS config

### Container Management (All Operations)
- `get_containers` - List containers
- `get_container_status` - Container status
- `get_container_config` - Container configuration
- `start_container` - Start container
- `stop_container` - Stop container
- `shutdown_container` - Graceful shutdown
- `reboot_container` - Reboot container
- `create_container` - Create container (basic)
- `create_container_advanced` - Advanced container creation
- `clone_container` - Clone container
- `update_container_config` - Update container config
- `delete_container` - Delete container
- `get_container_stats` - Container statistics

### Container Snapshots (Advanced)
- `create_container_snapshot` - Create container snapshot
- `list_container_snapshots` - List container snapshots
- `delete_container_snapshot` - Delete container snapshot
- `restore_container_snapshot` - Restore container snapshot

### Backups
- `list_backups` - List backups
- `create_vm_backup` - Create VM backup
- `restore_vm_backup` - Restore VM backup
- `delete_backup` - Delete backup
- `upload_backup` - Upload backup file

### Container Backups (Advanced)
- `create_container_backup` - Create container backup
- `restore_container_backup` - Restore container backup

### User Management (Write Operations)
- `create_user` - Create user
- `update_user` - Update user
- `delete_user` - Delete user
- `change_password` - Change password
- `create_group` - Create group
- `delete_group` - Delete group
- `create_role` - Create role
- `delete_role` - Delete role
- `set_acl` - Set ACL
- `create_api_token` - Create API token
- `delete_api_token` - Delete API token

### Storage Management (Write Operations)
- `create_storage` - Create storage
- `delete_storage` - Delete storage
- `update_storage` - Update storage
- `get_storage_quota` - Storage quota

### Resource Pools
- `list_pools` - List pools
- `get_pool` - Get pool
- `create_pool` - Create pool
- `update_pool` - Update pool
- `delete_pool` - Delete pool
- `get_pool_members` - Pool members

### High Availability
- `get_ha_status` - HA status
- `enable_ha_resource` - Enable HA
- `disable_ha_resource` - Disable HA

### Firewall & Network
- `get_firewall_rules` - Firewall rules
- `create_firewall_rule` - Create firewall rule
- `delete_firewall_rule` - Delete firewall rule
- `get_security_groups` - Security groups
- `create_security_group` - Create security group
- `get_network_interfaces` - Network interfaces
- `get_vlan_config` - VLAN config

### Statistics
- `get_node_stats` - Node statistics
- `get_vm_stats` - VM statistics (if not in default)
- `get_container_stats` - Container statistics (if not in default)

## Benefits

1. **Reduced LLM Confusion**: Fewer tools in the default set means LLMs are less likely to get confused about which tool to use
2. **Faster Tool Discovery**: Common operations are immediately available
3. **Security**: Destructive operations (node reboot, cluster changes) require explicit enablement
4. **Performance**: Smaller tool list means faster tool listing and selection

## Usage Examples

### Default Mode (Recommended for most users)
```bash
# No environment variables needed - uses default tools only
./proxmox-ve-mcp
```

### Full Mode (All tools enabled)
```bash
export MCP_ENABLE_ADVANCED_TOOLS=true
./proxmox-ve-mcp
```

Or:
```bash
export MCP_TOOLS_MODE=all
./proxmox-ve-mcp
```

## Tool Count

- **Default tools**: ~40-50 tools
- **Advanced tools**: ~60-70 tools
- **Total**: 107 tools

When advanced tools are disabled, LLMs only see the default set, making it much easier to choose the right tool for common operations.
