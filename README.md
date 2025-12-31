# Proxmox VE MCP

Model Context Protocol (MCP) server for Proxmox Virtual Environment infrastructure management. Control and monitor your Proxmox infrastructure through an AI-powered interface.

**Focused on:** Node management, VM lifecycle, container management, resource monitoring, and cluster operations.

## Features

- **8 management tools** for cluster and resource control
- **Cluster Management**: Monitor cluster health and node status
- **Virtual Machine Management**: List, monitor, and manage VMs
- **Container Management**: Manage LXC containers
- **Node Monitoring**: Track resource usage and uptime
- **Stdio Transport**: MCP protocol over standard input/output for seamless integration

## Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/surrealwolf/proxmox-ve-mcp.git
cd proxmox-ve-mcp
go build -o bin/proxmox-ve-mcp ./cmd
```

### Configuration

Create a `.env` file from the example:

```bash
cp .env.example .env
```

Then edit `.env` with your Proxmox credentials:

```bash
PROXMOX_BASE_URL=https://your-proxmox-server.com:8006
PROXMOX_API_TOKEN=user@realm!tokenid=token-secret-here
PROXMOX_SKIP_SSL_VERIFY=false
LOG_LEVEL=info
```

### Obtaining API Token

1. Log in to Proxmox Web UI
2. Go to Datacenter → Permissions → API Tokens
3. Create a new API token with appropriate permissions
4. The token format is: `user@realm!tokenid=secret`

### Running the Server

```bash
./bin/proxmox-ve-mcp
```

The server listens on stdio and is ready for MCP protocol messages.

## Available Tools (8 Total)

### Cluster & Node Management (2 tools)
- `get_nodes` - List all nodes in the Proxmox cluster
- `get_node_status` - Get detailed status for a specific node

### Virtual Machine Management (2 tools)
- `get_vms` - List all VMs on a specific node
- `get_vm_status` - Get detailed VM information and status

### Container Management (2 tools)
- `get_containers` - List all containers on a specific node
- `get_container_status` - Get detailed container information and status

### Cluster Operations (2 tools)
- [Planned] Backup operations
- [Planned] Resource allocation

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PROXMOX_BASE_URL` | Proxmox server URL with port | Required |
| `PROXMOX_API_TOKEN` | API token from Proxmox | Required |
| `PROXMOX_SKIP_SSL_VERIFY` | Skip SSL certificate verification | false |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info |

## API Reference

For detailed Proxmox API documentation, see: https://pve.proxmox.com/pve-docs/api-viewer/index.html

## Development

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Docker

```bash
make docker-build
make docker-run
```

## License

MIT License - See LICENSE file for details

## Support

For issues and questions:
- Check the [Proxmox API Documentation](https://pve.proxmox.com/pve-docs/api-viewer/index.html)
- Review implementation examples in `internal/`
