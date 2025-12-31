---
name: container-management
description: Create, manage, and optimize LXC containers in Proxmox. Control container lifecycle, manage resources, and coordinate container deployments across nodes.
---

# Container Management Skill

Create, manage, and optimize LXC containers in your Proxmox environment.

## What this skill does

This skill enables you to:
- List containers on specific nodes
- Get detailed container configuration and status
- Start, stop, and reboot containers
- Create new LXC containers
- Modify container resource allocation
- Monitor container performance metrics
- Manage container templates
- Plan container deployment strategies
- Optimize resource allocation for containers

## When to use this skill

Use this skill when you need to:
- Check container status and configuration
- Manage container lifecycle (start/stop/reboot)
- Monitor container performance and resource usage
- Adjust container resources (CPU, memory, storage)
- Create new containers
- Troubleshoot container issues
- Plan container migrations
- Optimize container placement
- Manage container templates

## Available Tools

- `get_containers` - List all containers on a specific node
- `get_container_status` - Get detailed container status and configuration
- `start_container` - Start a container
- `stop_container` - Stop a container immediately
- `shutdown_container` - Gracefully shutdown a container
- `reboot_container` - Reboot a container
- `create_container` - Create a new LXC container
- `resize_container` - Adjust container resources
- `delete_container` - Delete a container
- `create_container_snapshot` - Create container snapshot
- `restore_container_snapshot` - Restore from snapshot

## Typical Workflows

### Container Lifecycle Management
1. Use `get_containers` to list available containers
2. Use `get_container_status` to check container state
3. Use start/stop/reboot to manage container operations
4. Monitor container health during changes

### Container Creation & Deployment
1. Use `create_container` to provision new container
2. Use `get_container_status` to verify configuration
3. Use `resize_container` to adjust resources as needed
4. Document container details for reference

### Container Optimization
1. Use `get_container_status` to analyze resource usage
2. Use `resize_container` to optimize allocation
3. Use snapshots for testing before changes
4. Monitor performance after optimization

### Container Troubleshooting
1. Use `get_container_status` to diagnose issues
2. Use reboot/restart to recover from problems
3. Use snapshots to rollback problematic changes
4. Analyze logs and metrics for root cause

## Example Questions

- "List all containers on the worker node"
- "What's the status and resource usage of container 101?"
- "Start the database container"
- "Create a new container with 2 cores and 4GB RAM"
- "Resize container 102 to have more memory"
- "Create a snapshot before the application update"
- "Show me all containers and their resource allocation"

## Response Format

When using this skill, I provide:
- Container listings with status and resources
- Detailed container configuration and metrics
- Status confirmations for container operations
- Resource utilization analysis
- Optimization recommendations

## Best Practices

- Monitor container performance regularly
- Use snapshots before major changes
- Plan resource allocation carefully
- Balance containers across nodes
- Implement monitoring for critical containers
- Use container templates for consistency
- Document container configuration
- Test changes in development first
- Use graceful shutdown when possible
- Monitor disk usage and resource limits
- Clean up unused containers regularly

