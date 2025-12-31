package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetContainers retrieves a list of all containers on a specific node
func (c *Client) GetContainers(ctx context.Context, nodeName string) ([]Container, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/lxc", nodeName), nil)
	if err != nil {
		return nil, err
	}

	var containers []Container
	if err := json.Unmarshal(marshalJSON(data), &containers); err != nil {
		return nil, fmt.Errorf("failed to parse containers: %w", err)
	}

	return containers, nil
}

// GetContainer retrieves information about a specific container
func (c *Client) GetContainer(ctx context.Context, nodeName string, containerID int) (*Container, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/lxc/%d/status/current", nodeName, containerID), nil)
	if err != nil {
		return nil, err
	}

	var container Container
	if err := json.Unmarshal(marshalJSON(data), &container); err != nil {
		return nil, fmt.Errorf("failed to parse container: %w", err)
	}

	return &container, nil
}

// GetContainerConfig retrieves the full configuration of a container
func (c *Client) GetContainerConfig(ctx context.Context, nodeName string, containerID int) (map[string]interface{}, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/lxc/%d/config", nodeName, containerID), nil)
	if err != nil {
		return nil, err
	}

	config, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected container config format")
	}

	return config, nil
}

// StartContainer starts an LXC container
func (c *Client) StartContainer(ctx context.Context, nodeName string, containerID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc/%d/status/start", nodeName, containerID), nil)
}

// StopContainer stops an LXC container
func (c *Client) StopContainer(ctx context.Context, nodeName string, containerID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc/%d/status/stop", nodeName, containerID), nil)
}

// ShutdownContainer gracefully shuts down an LXC container
func (c *Client) ShutdownContainer(ctx context.Context, nodeName string, containerID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc/%d/status/shutdown", nodeName, containerID), nil)
}

// RebootContainer reboots an LXC container
func (c *Client) RebootContainer(ctx context.Context, nodeName string, containerID int) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc/%d/status/reboot", nodeName, containerID), nil)
}

// CreateContainer creates a new LXC container with basic configuration
func (c *Client) CreateContainer(ctx context.Context, nodeName string, config map[string]interface{}) (interface{}, error) {
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc", nodeName), config)
}

// CreateContainerFull creates a new LXC container with full configuration
func (c *Client) CreateContainerFull(ctx context.Context, nodeName string, containerID int, hostname string, storage string, memory int, cores int, ostype string) (interface{}, error) {
	config := map[string]interface{}{
		"vmid":     containerID,
		"hostname": hostname,
		"storage":  storage,
		"memory":   memory,
		"cores":    cores,
		"ostype":   ostype,
	}
	return c.CreateContainer(ctx, nodeName, config)
}

// DeleteContainer deletes an LXC container
func (c *Client) DeleteContainer(ctx context.Context, nodeName string, containerID int, force bool) (interface{}, error) {
	data := map[string]interface{}{
		"force": force,
	}
	return c.doRequest(ctx, "DELETE", fmt.Sprintf("nodes/%s/lxc/%d", nodeName, containerID), data)
}

// CloneContainer clones an existing LXC container
func (c *Client) CloneContainer(ctx context.Context, nodeName string, sourceContainerID int, newContainerID int, newHostname string, full bool) (interface{}, error) {
	config := map[string]interface{}{
		"newid":    newContainerID,
		"hostname": newHostname,
		"full":     full,
	}
	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc/%d/clone", nodeName, sourceContainerID), config)
}

// UpdateContainer updates a container's configuration
func (c *Client) UpdateContainer(ctx context.Context, nodeName string, containerID int, config map[string]interface{}) (interface{}, error) {
	return c.doRequest(ctx, "PUT", fmt.Sprintf("nodes/%s/lxc/%d/config", nodeName, containerID), config)
}
