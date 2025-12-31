package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetNodes retrieves a list of all nodes in the cluster
func (c *Client) GetNodes(ctx context.Context) ([]Node, error) {
	data, err := c.doRequest(ctx, "GET", "nodes", nil)
	if err != nil {
		return nil, err
	}

	var nodes []Node
	if err := json.Unmarshal(marshalJSON(data), &nodes); err != nil {
		return nil, fmt.Errorf("failed to parse nodes: %w", err)
	}

	return nodes, nil
}

// GetNode retrieves information about a specific node
func (c *Client) GetNode(ctx context.Context, nodeName string) (*NodeStatus, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/status", nodeName), nil)
	if err != nil {
		return nil, err
	}

	var nodeStatus NodeStatus
	if err := json.Unmarshal(marshalJSON(data), &nodeStatus); err != nil {
		return nil, fmt.Errorf("failed to parse node status: %w", err)
	}

	return &nodeStatus, nil
}

// GetClusterStatus retrieves cluster-wide status information
func (c *Client) GetClusterStatus(ctx context.Context) (interface{}, error) {
	data, err := c.doRequest(ctx, "GET", "cluster/status", nil)
	if err != nil {
		return nil, err
	}

	// cluster/status returns an array of nodes
	return data, nil
}

// GetStorage retrieves a list of all storage devices in the cluster
func (c *Client) GetStorage(ctx context.Context) ([]Storage, error) {
	data, err := c.doRequest(ctx, "GET", "storage", nil)
	if err != nil {
		return nil, err
	}

	var storage []Storage
	if err := json.Unmarshal(marshalJSON(data), &storage); err != nil {
		return nil, fmt.Errorf("failed to parse storage: %w", err)
	}

	return storage, nil
}

// GetNodeStorage retrieves storage devices for a specific node
func (c *Client) GetNodeStorage(ctx context.Context, nodeName string) ([]Storage, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/storage", nodeName), nil)
	if err != nil {
		return nil, err
	}

	var storage []Storage
	if err := json.Unmarshal(marshalJSON(data), &storage); err != nil {
		return nil, fmt.Errorf("failed to parse node storage: %w", err)
	}

	return storage, nil
}

// GetTasks retrieves a list of background tasks
func (c *Client) GetTasks(ctx context.Context) ([]Task, error) {
	data, err := c.doRequest(ctx, "GET", "cluster/tasks", nil)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	if err := json.Unmarshal(marshalJSON(data), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks: %w", err)
	}

	return tasks, nil
}

// GetClusterResources retrieves an overview of cluster resources (nodes, VMs, containers, storage)
func (c *Client) GetClusterResources(ctx context.Context) (interface{}, error) {
	data, err := c.doRequest(ctx, "GET", "cluster/resources", nil)
	if err != nil {
		return nil, err
	}

	return data, nil
}
