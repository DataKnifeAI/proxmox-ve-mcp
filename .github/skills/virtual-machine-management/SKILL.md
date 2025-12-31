---
name: virtual-machine-management
description: Create, manage, and optimize virtual machines in Proxmox. Control VM lifecycle, monitor performance, adjust resources, and plan VM deployment strategies.
---

# Virtual Machine Management Skill

Create, manage, and optimize virtual machines in your Proxmox environment.

## What this skill does

This skill enables you to:
- List virtual machines on specific nodes
- Get detailed VM configuration and status
- Start, stop, and reboot virtual machines
- Create new virtual machines
- Modify VM resource allocation
- Monitor VM performance metrics
- Manage VM snapshots
- Plan VM deployment strategies
- Optimize resource allocation

## When to use this skill

Use this skill when you need to:
- Check VM status and configuration
- Manage VM lifecycle (start/stop/reboot)
- Monitor VM performance and resource usage
- Adjust VM resources (CPU, memory, storage)
- Create new virtual machines
- Troubleshoot VM issues
- Plan VM migrations
- Optimize VM placement

## Available Tools

- `get_vms` - List all VMs on a specific node
- `get_vm_status` - Get detailed VM status and configuration
- `start_vm` - Start a virtual machine
- `stop_vm` - Stop a VM immediately
- `shutdown_vm` - Gracefully shutdown a VM
- `reboot_vm` - Reboot a virtual machine
- `create_vm` - Create a new virtual machine
- `resize_vm` - Adjust VM resources (CPU, memory)
- `delete_vm` - Delete a virtual machine
- `create_vm_snapshot` - Create VM snapshot
- `restore_vm_snapshot` - Restore from snapshot

## Typical Workflows

### VM Lifecycle Management
1. Use `get_vms` to list available VMs
2. Use `get_vm_status` to check VM state
3. Use start/stop/reboot to manage VM operations
4. Monitor VM health during changes

### VM Creation & Configuration
1. Use `create_vm` to provision new VM
2. Use `get_vm_status` to verify configuration
3. Use `resize_vm` to adjust resources as needed
4. Document VM details for reference

### VM Optimization
1. Use `get_vm_status` to analyze current resource usage
2. Use `resize_vm` to optimize CPU/memory allocation
3. Use snapshots for testing before major changes
4. Monitor performance improvements

### VM Troubleshooting
1. Use `get_vm_status` to check VM health
2. Use reboot/restart to recover from issues
3. Use snapshots to rollback problematic changes
4. Analyze performance metrics for root cause

## Example Questions

- "List all VMs on the production node"
- "What's the status and resource usage of VM 100?"
- "Start the web server VM"
- "Create a new VM with 4 cores and 8GB RAM"
- "Resize VM 200 to have more CPU cores"
- "Create a snapshot before the update"
- "Show me all VMs and their resource allocation"

## Response Format

When using this skill, I provide:
- VM listings with status and resource allocation
- Detailed VM configuration and performance metrics
- Status confirmations for VM operations
- Resource utilization analysis
- Optimization recommendations

## Best Practices

- Monitor VM performance regularly
- Use snapshots before major changes
- Plan resource allocation carefully
- Balance VMs across nodes
- Implement high-availability for critical VMs
- Keep VM templates updated
- Document VM configuration
- Test changes in non-production first
- Use graceful shutdown when possible
- Monitor disk usage to prevent full disks

