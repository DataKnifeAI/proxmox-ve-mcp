package proxmox

import (
	"context"
	"fmt"
)

// CreateVMSnapshot creates a snapshot of a virtual machine
func (c *Client) CreateVMSnapshot(ctx context.Context, nodeName string, vmID int, snapName string, description string) (interface{}, error) {
	data := map[string]string{
		"snapname": snapName,
	}
	if description != "" {
		data["description"] = description
	}

	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/snapshot", nodeName, vmID), data)
}

// ListVMSnapshots lists all snapshots for a virtual machine
func (c *Client) ListVMSnapshots(ctx context.Context, nodeName string, vmID int) ([]map[string]interface{}, error) {
	result, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/qemu/%d/snapshot", nodeName, vmID), nil)
	if err != nil {
		return []map[string]interface{}{}, err
	}
	if result == nil {
		return []map[string]interface{}{}, nil
	}

	// Handle response as slice
	snapshots := []map[string]interface{}{}
	if data, ok := result.([]interface{}); ok {
		for _, item := range data {
			if snap, ok := item.(map[string]interface{}); ok {
				snapshots = append(snapshots, snap)
			}
		}
	}
	return snapshots, nil
}

// DeleteVMSnapshot deletes a snapshot from a virtual machine
func (c *Client) DeleteVMSnapshot(ctx context.Context, nodeName string, vmID int, snapName string, force bool) (interface{}, error) {
	data := map[string]interface{}{
		"force": force,
	}

	return c.doRequest(ctx, "DELETE", fmt.Sprintf("nodes/%s/qemu/%d/snapshot/%s", nodeName, vmID, snapName), data)
}

// RestoreVMSnapshot restores a virtual machine from a snapshot
func (c *Client) RestoreVMSnapshot(ctx context.Context, nodeName string, vmID int, snapName string) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/snapshot/%s/rollback", nodeName, vmID, snapName), nil)
}

// GetVMFirewallRules retrieves firewall rules for a virtual machine
func (c *Client) GetVMFirewallRules(ctx context.Context, nodeName string, vmID int) ([]map[string]interface{}, error) {
	result, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/qemu/%d/firewall/rules", nodeName, vmID), nil)
	if err != nil {
		return []map[string]interface{}{}, err
	}
	if result == nil {
		return []map[string]interface{}{}, nil
	}

	// Handle response as slice
	rules := []map[string]interface{}{}
	if data, ok := result.([]interface{}); ok {
		for _, item := range data {
			if rule, ok := item.(map[string]interface{}); ok {
				rules = append(rules, rule)
			}
		}
	}
	return rules, nil
}

// MigrateVM migrates a virtual machine to another node
func (c *Client) MigrateVM(ctx context.Context, nodeName string, vmID int, targetNode string, online bool) (interface{}, error) {
	config := map[string]interface{}{
		"target": targetNode,
	}
	if online {
		config["online"] = 1
	}

	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/migrate", nodeName, vmID), config)
}

// CreateContainerSnapshot creates a snapshot of a container
func (c *Client) CreateContainerSnapshot(ctx context.Context, nodeName string, containerID int, snapName string, description string) (interface{}, error) {
	data := map[string]string{
		"snapname": snapName,
	}
	if description != "" {
		data["description"] = description
	}

	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc/%d/snapshot", nodeName, containerID), data)
}

// ListContainerSnapshots lists all snapshots for a container
func (c *Client) ListContainerSnapshots(ctx context.Context, nodeName string, containerID int) ([]map[string]interface{}, error) {
	result, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/lxc/%d/snapshot", nodeName, containerID), nil)
	if err != nil {
		return []map[string]interface{}{}, err
	}
	if result == nil {
		return []map[string]interface{}{}, nil
	}

	// Handle response as slice
	snapshots := []map[string]interface{}{}
	if data, ok := result.([]interface{}); ok {
		for _, item := range data {
			if snap, ok := item.(map[string]interface{}); ok {
				snapshots = append(snapshots, snap)
			}
		}
	}
	return snapshots, nil
}

// DeleteContainerSnapshot deletes a snapshot from a container
func (c *Client) DeleteContainerSnapshot(ctx context.Context, nodeName string, containerID int, snapName string, force bool) (interface{}, error) {
	data := map[string]interface{}{
		"force": force,
	}

	return c.doRequest(ctx, "DELETE", fmt.Sprintf("nodes/%s/lxc/%d/snapshot/%s", nodeName, containerID, snapName), data)
}

// RestoreContainerSnapshot restores a container from a snapshot
func (c *Client) RestoreContainerSnapshot(ctx context.Context, nodeName string, containerID int, snapName string) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc/%d/snapshot/%s/rollback", nodeName, containerID, snapName), nil)
}
