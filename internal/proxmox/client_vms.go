package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetVMs retrieves a list of all VMs on a specific node
func (c *Client) GetVMs(ctx context.Context, nodeName string) ([]VM, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/qemu", nodeName), nil)
	if err != nil {
		return nil, err
	}

	var vms []VM
	if err := json.Unmarshal(marshalJSON(data), &vms); err != nil {
		return nil, fmt.Errorf("failed to parse VMs: %w", err)
	}

	return vms, nil
}

// GetVM retrieves information about a specific VM
func (c *Client) GetVM(ctx context.Context, nodeName string, vmID int) (*VM, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/qemu/%d/status/current", nodeName, vmID), nil)
	if err != nil {
		return nil, err
	}

	var vm VM
	if err := json.Unmarshal(marshalJSON(data), &vm); err != nil {
		return nil, fmt.Errorf("failed to parse VM: %w", err)
	}

	return &vm, nil
}

// GetVMConfig retrieves the full configuration of a virtual machine
func (c *Client) GetVMConfig(ctx context.Context, nodeName string, vmID int) (map[string]interface{}, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/qemu/%d/config", nodeName, vmID), nil)
	if err != nil {
		return nil, err
	}

	config, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected VM config format")
	}

	return config, nil
}

// StartVM powers on a virtual machine
func (c *Client) StartVM(ctx context.Context, nodeName string, vmID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/status/start", nodeName, vmID), nil)
}

// StopVM powers off a virtual machine
func (c *Client) StopVM(ctx context.Context, nodeName string, vmID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/status/stop", nodeName, vmID), nil)
}

// RebootVM reboots a virtual machine
func (c *Client) RebootVM(ctx context.Context, nodeName string, vmID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/status/reboot", nodeName, vmID), nil)
}

// ShutdownVM gracefully shuts down a virtual machine
func (c *Client) ShutdownVM(ctx context.Context, nodeName string, vmID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/status/shutdown", nodeName, vmID), nil)
}

// SuspendVM suspends (pauses) a virtual machine
func (c *Client) SuspendVM(ctx context.Context, nodeName string, vmID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/status/suspend", nodeName, vmID), nil)
}

// ResumeVM resumes a suspended virtual machine
func (c *Client) ResumeVM(ctx context.Context, nodeName string, vmID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/status/resume", nodeName, vmID), nil)
}

// CreateVM creates a new virtual machine with basic configuration
func (c *Client) CreateVM(ctx context.Context, nodeName string, config map[string]interface{}) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu", nodeName), config)
}

// CreateVMFull creates a new virtual machine with full configuration
func (c *Client) CreateVMFull(ctx context.Context, nodeName string, vmID int, name string, memory int, cores int, sockets int) (interface{}, error) {
	config := map[string]interface{}{
		"vmid":    vmID,
		"name":    name,
		"memory":  memory,
		"cores":   cores,
		"sockets": sockets,
	}
	return c.CreateVM(ctx, nodeName, config)
}

// DeleteVM deletes a virtual machine
func (c *Client) DeleteVM(ctx context.Context, nodeName string, vmID int, force bool) (interface{}, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d", nodeName, vmID)
	if force {
		endpoint += "?force=1"
	}
	return c.doRequest(ctx, "DELETE", endpoint, nil)
}

// CloneVM clones an existing virtual machine
func (c *Client) CloneVM(ctx context.Context, nodeName string, sourceVMID int, newVMID int, newName string, full bool) (interface{}, error) {
	config := map[string]interface{}{
		"newid": newVMID,
		"name":  newName,
		"full":  full,
	}
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/clone", nodeName, sourceVMID), config)
}

// UpdateVM updates a virtual machine's configuration
func (c *Client) UpdateVM(ctx context.Context, nodeName string, vmID int, config map[string]interface{}) (interface{}, error) {
	return c.doRequest(ctx, "PUT", fmt.Sprintf("nodes/%s/qemu/%d/config", nodeName, vmID), config)
}

// GetVMConsole gets console access information for a VM
func (c *Client) GetVMConsole(ctx context.Context, nodeName string, vmID int) (map[string]interface{}, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/qemu/%d/status/current", nodeName, vmID), nil)
	if err != nil {
		return nil, err
	}

	console, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected console data format")
	}

	return console, nil
}
