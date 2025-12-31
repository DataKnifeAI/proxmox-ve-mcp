package proxmox

import (
	"context"
	"fmt"
)

// GetStorageInfo retrieves detailed information about a specific storage device
func (c *Client) GetStorageInfo(ctx context.Context, storage string) (map[string]interface{}, error) {
	data, err := c.doRequest(ctx, "GET", fmt.Sprintf("storage/%s", storage), nil)
	if err != nil {
		return nil, err
	}

	info, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected storage info format")
	}

	return info, nil
}

// GetStorageContent lists the contents of a storage device (ISOs, backups, templates, etc.)
func (c *Client) GetStorageContent(ctx context.Context, storage string) ([]map[string]interface{}, error) {
	result, err := c.doRequest(ctx, "GET", fmt.Sprintf("storage/%s/content", storage), nil)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []map[string]interface{}{}, nil
	}

	content := []map[string]interface{}{}
	if data, ok := result.([]interface{}); ok {
		for _, item := range data {
			if content_item, ok := item.(map[string]interface{}); ok {
				content = append(content, content_item)
			}
		}
	}
	return content, nil
}

// CreateStorage creates a new storage mount
func (c *Client) CreateStorage(ctx context.Context, storage, storageType, content string, config map[string]interface{}) (interface{}, error) {
	body := map[string]interface{}{
		"storage": storage,
		"type":    storageType,
		"content": content,
	}
	// Merge additional config
	for k, v := range config {
		body[k] = v
	}

	return c.doRequest(ctx, "POST", "storage", body)
}

// DeleteStorage removes a storage configuration
func (c *Client) DeleteStorage(ctx context.Context, storage string) (interface{}, error) {
	return c.doRequest(ctx, "DELETE", fmt.Sprintf("storage/%s", storage), nil)
}

// UpdateStorage modifies storage configuration
func (c *Client) UpdateStorage(ctx context.Context, storage string, config map[string]interface{}) (interface{}, error) {
	body := config
	body["storage"] = storage
	return c.doRequest(ctx, "PUT", fmt.Sprintf("storage/%s", storage), body)
}

// GetStorageQuota retrieves storage quota information
func (c *Client) GetStorageQuota(ctx context.Context, storage string) (map[string]interface{}, error) {
	// Proxmox doesn't have a dedicated quota endpoint, but we can get info and content
	info, err := c.GetStorageInfo(ctx, storage)
	if err != nil {
		return nil, err
	}

	// Try to get content to calculate usage
	content, err := c.GetStorageContent(ctx, storage)
	if err != nil {
		return nil, err
	}

	var totalSize int64
	for _, item := range content {
		if size, ok := item["size"].(float64); ok {
			totalSize += int64(size)
		}
	}

	quota := map[string]interface{}{
		"storage":    storage,
		"info":       info,
		"used_bytes": totalSize,
		"content":    content,
	}

	return quota, nil
}

// UploadBackup uploads a backup file to storage (placeholder implementation)
func (c *Client) UploadBackup(ctx context.Context, storage, backupID string, filePath string) (interface{}, error) {
	// Note: This is a placeholder. Real implementation would require file upload handling
	// For now, return an error indicating this operation requires direct file upload
	return nil, fmt.Errorf("backup upload requires direct HTTP file upload - not yet fully implemented via REST API")
}
