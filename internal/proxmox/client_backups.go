package proxmox

import (
	"context"
	"fmt"
)

// Backup represents a backup file
type Backup struct {
	BackupID  string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	VMID      int    `json:"vmid,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Notes     string `json:"notes,omitempty"`
	CTime     int64  `json:"ctime,omitempty"`
	Content   string `json:"content,omitempty"`
	Verified  int    `json:"verified,omitempty"`
	Encrypted int    `json:"encrypted,omitempty"`
	Nodes     string `json:"nodes,omitempty"`
}

// CreateVMBackup creates a backup of a virtual machine
func (c *Client) CreateVMBackup(ctx context.Context, nodeName string, vmID int, storage, backupID, notes string) (interface{}, error) {
	body := map[string]interface{}{
		"storage": storage,
		"vmid":    vmID,
	}
	if backupID != "" {
		body["id"] = backupID
	}
	if notes != "" {
		body["notes"] = notes
	}

	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu/%d/backup", nodeName, vmID), body)
}

// CreateContainerBackup creates a backup of a container
func (c *Client) CreateContainerBackup(ctx context.Context, nodeName string, containerID int, storage, backupID, notes string) (interface{}, error) {
	body := map[string]interface{}{
		"storage": storage,
		"vmid":    containerID,
	}
	if backupID != "" {
		body["id"] = backupID
	}
	if notes != "" {
		body["notes"] = notes
	}

	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc/%d/backup", nodeName, containerID), body)
}

// ListBackups returns available backups in storage across all nodes
func (c *Client) ListBackups(ctx context.Context, storage string) ([]Backup, error) {
	// Get all nodes first
	nodes, err := c.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %v", err)
	}

	var allBackups []Backup

	// Try to get backups from each node's storage
	for _, node := range nodes {
		data, err := c.doRequest(ctx, "GET", fmt.Sprintf("nodes/%s/storage/%s/content", node.Node, storage), nil)
		if err != nil {
			// Log error but continue with other nodes
			c.logger.Warnf("Failed to list backups from node %s: %v", node.Node, err)
			continue
		}

		backups := []Backup{}
		if err := c.unmarshalData(data, &backups); err != nil {
			c.logger.Warnf("Failed to unmarshal backups from node %s: %v", node.Node, err)
			continue
		}

		allBackups = append(allBackups, backups...)
	}

	return allBackups, nil
}

// DeleteBackup removes a backup file from a specific node's storage
func (c *Client) DeleteBackup(ctx context.Context, storage, backupID string) (interface{}, error) {
	// Get all nodes to find which one has the backup
	nodes, err := c.GetNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %v", err)
	}

	var lastErr error

	// Try to delete backup from each node
	for _, node := range nodes {
		result, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("nodes/%s/storage/%s/content/%s", node.Node, storage, backupID), nil)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("backup not found on any node")
}

// RestoreVMBackup restores a VM from a backup
func (c *Client) RestoreVMBackup(ctx context.Context, nodeName string, backupID, storage string) (interface{}, error) {
	body := map[string]interface{}{
		"archive": backupID,
		"storage": storage,
	}

	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/qemu", nodeName), body)
}

// RestoreContainerBackup restores a container from a backup
func (c *Client) RestoreContainerBackup(ctx context.Context, nodeName string, backupID, storage string) (interface{}, error) {
	body := map[string]interface{}{
		"archive": backupID,
		"storage": storage,
	}

	return c.doRequest(ctx, "POST", fmt.Sprintf("nodes/%s/lxc", nodeName), body)
}
