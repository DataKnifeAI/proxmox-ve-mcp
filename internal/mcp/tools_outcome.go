package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// outcomeBasedToolDefs returns the consolidated tool set: one outcome per tool, flat params. Total under 30.
func (s *Server) outcomeBasedToolDefs() []ToolDefinition {
	return []ToolDefinition{
		{"get_cluster", "Get cluster overview: nodes, resources, and status in one call", s.getCluster, map[string]any{}, CategoryDefault},
		{"get_node", "Get one node's details. Use detail: status, config, disks, cert, logs, network, dns, stats, apt", s.getNode, map[string]any{
			"node_name": map[string]any{"type": "string", "description": "Node name"},
			"detail":    map[string]any{"type": "string", "description": "status, config, disks, cert, logs, network, dns, stats, apt (default: status)"},
		}, CategoryDefault},
		{"manage_storage", "Storage outcome: list, info, content, create, update, delete, quota. Use action and flat params.", s.manageStorage, map[string]any{
			"action":       map[string]any{"type": "string", "description": "list, info, content, create, update, delete, quota"},
			"node_name":    map[string]any{"type": "string", "description": "Node (optional for list)"},
			"storage":      map[string]any{"type": "string", "description": "Storage ID (for info, content, create, update, delete, quota)"},
			"storage_type": map[string]any{"type": "string", "description": "For create: dir, nfs, lvm, iscsi, etc."},
			"content":      map[string]any{"type": "string", "description": "For create: content types (images, rootdir, backups, etc.)"},
			"confirm":      map[string]any{"type": "string", "description": "For delete: set to storage ID to confirm"},
		}, CategoryDefault},
		{"list_guests", "List VMs and/or containers on a node or cluster-wide", s.listGuests, map[string]any{
			"node_name": map[string]any{"type": "string", "description": "Node (optional; omit for all)"},
			"type":      map[string]any{"type": "string", "description": "vm, lxc, or all (default: all)"},
		}, CategoryDefault},
		{"get_guest", "Get one guest's details. Use detail: status, config, console, firewall, stats", s.getGuest, map[string]any{
			"node_name":  map[string]any{"type": "string", "description": "Node name"},
			"guest_type": map[string]any{"type": "string", "description": "vm or lxc"},
			"vmid":       map[string]any{"type": "integer", "description": "VM or container ID"},
			"detail":     map[string]any{"type": "string", "description": "status, config, console, firewall, stats (default: status)"},
		}, CategoryDefault},
		{"control_guest", "Guest lifecycle: start, stop, shutdown, reboot, suspend, resume, migrate", s.controlGuest, map[string]any{
			"node_name":   map[string]any{"type": "string", "description": "Node name"},
			"guest_type":  map[string]any{"type": "string", "description": "vm or lxc"},
			"vmid":        map[string]any{"type": "integer", "description": "VM or container ID"},
			"action":      map[string]any{"type": "string", "description": "start, stop, shutdown, reboot, suspend, resume, migrate"},
			"target_node": map[string]any{"type": "string", "description": "For action=migrate: target node"},
			"online":      map[string]any{"type": "boolean", "description": "For migrate: live migration (optional)"},
		}, CategoryDefault},
		{"create_guest", "Create a VM or container. Use guest_type and flat params (name/hostname, memory, cores, storage, net0, etc.)", s.createGuest, map[string]any{
			"guest_type": map[string]any{"type": "string", "description": "vm or lxc"},
			"node_name":  map[string]any{"type": "string", "description": "Node name"},
			"vmid":       map[string]any{"type": "integer", "description": "VM/container ID (unique)"},
			"name":       map[string]any{"type": "string", "description": "VM name (optional)"},
			"hostname":   map[string]any{"type": "string", "description": "Container hostname (optional)"},
			"memory":     map[string]any{"type": "integer", "description": "Memory MB (optional)"},
			"cores":      map[string]any{"type": "integer", "description": "CPU cores (optional)"},
			"sockets":    map[string]any{"type": "integer", "description": "CPU sockets, VM only (optional)"},
			"storage":    map[string]any{"type": "string", "description": "Storage ID (optional; recommended for lxc)"},
			"ostype":     map[string]any{"type": "string", "description": "OS type for container (optional)"},
			"ide2":       map[string]any{"type": "string", "description": "CD/DVD, VM only (optional)"},
			"sata0":      map[string]any{"type": "string", "description": "Primary disk, VM (optional)"},
			"net0":       map[string]any{"type": "string", "description": "Network (optional)"},
			"rootfs":     map[string]any{"type": "string", "description": "Root FS, container (optional)"},
		}, CategoryDefault},
		{"update_guest", "Update VM or container config (flat params: memory, cores, name, hostname, template, net0, sockets)", s.updateGuestConfig, map[string]any{
			"node_name":  map[string]any{"type": "string", "description": "Node name"},
			"guest_type": map[string]any{"type": "string", "description": "vm or lxc"},
			"vmid":       map[string]any{"type": "integer", "description": "VM or container ID"},
			"memory":     map[string]any{"type": "integer", "description": "Memory MB (optional)"},
			"cores":      map[string]any{"type": "integer", "description": "CPU cores (optional)"},
			"name":       map[string]any{"type": "string", "description": "VM name (optional)"},
			"hostname":   map[string]any{"type": "string", "description": "Container hostname (optional)"},
			"template":   map[string]any{"type": "integer", "description": "1 to mark VM as template (optional)"},
			"net0":       map[string]any{"type": "string", "description": "Network (optional)"},
			"sockets":    map[string]any{"type": "integer", "description": "CPU sockets (optional)"},
		}, CategoryDefault},
		{"clone_guest", "Clone a VM or container", s.cloneGuest, map[string]any{
			"node_name":   map[string]any{"type": "string", "description": "Node name"},
			"guest_type":  map[string]any{"type": "string", "description": "vm or lxc"},
			"source_vmid": map[string]any{"type": "integer", "description": "Source ID"},
			"new_vmid":    map[string]any{"type": "integer", "description": "New ID (unique)"},
			"new_name":    map[string]any{"type": "string", "description": "New name or hostname"},
			"full":        map[string]any{"type": "boolean", "description": "Full clone (default true)"},
		}, CategoryDefault},
		{"delete_guest", "Delete a VM or container. Requires confirm set to '<node_name>/<vmid>'", s.deleteGuest, map[string]any{
			"guest_type": map[string]any{"type": "string", "description": "vm or lxc"},
			"node_name":  map[string]any{"type": "string", "description": "Node name"},
			"vmid":       map[string]any{"type": "integer", "description": "VM or container ID"},
			"confirm":    map[string]any{"type": "string", "description": "Required: '<node_name>/<vmid>' to confirm"},
			"force":      map[string]any{"type": "boolean", "description": "Force if running (optional)"},
		}, CategoryDefault},
		{"manage_snapshot", "Snapshot outcome: create, list, delete, restore. Use action; confirm required for delete.", s.manageSnapshot, map[string]any{
			"action":      map[string]any{"type": "string", "description": "create, list, delete, restore"},
			"node_name":   map[string]any{"type": "string", "description": "Node name"},
			"guest_type":  map[string]any{"type": "string", "description": "vm or lxc"},
			"vmid":        map[string]any{"type": "integer", "description": "VM or container ID"},
			"snap_name":   map[string]any{"type": "string", "description": "Snapshot name (create/delete/restore)"},
			"description": map[string]any{"type": "string", "description": "Description (create, optional)"},
			"confirm":    map[string]any{"type": "string", "description": "For delete: '<node_name>/<vmid>/<snap_name>'"},
			"force":      map[string]any{"type": "boolean", "description": "For delete (optional)"},
		}, CategoryDefault},
		{"list_backups", "List backups in a storage", s.listBackups, map[string]any{
			"storage": map[string]any{"type": "string", "description": "Storage ID"},
		}, CategoryDefault},
		{"manage_backup", "Backup outcome: create, restore, or delete. Use action; confirm required for delete.", s.manageBackup, map[string]any{
			"action":     map[string]any{"type": "string", "description": "create, restore, delete"},
			"guest_type": map[string]any{"type": "string", "description": "vm or lxc"},
			"node_name":  map[string]any{"type": "string", "description": "Node name"},
			"vmid":       map[string]any{"type": "integer", "description": "VM/container ID (create/restore)"},
			"backup_id":  map[string]any{"type": "string", "description": "Backup ID (restore/delete)"},
			"storage":   map[string]any{"type": "string", "description": "Storage ID"},
			"notes":     map[string]any{"type": "string", "description": "Notes (create, optional)"},
			"confirm":   map[string]any{"type": "string", "description": "For delete: '<storage>/<backup_id>'"},
		}, CategoryDefault},
		{"manage_tasks", "Task outcome: list (by node or cluster), status, log, or cancel", s.manageTasks, map[string]any{
			"action":    map[string]any{"type": "string", "description": "list, status, log, cancel"},
			"node_name": map[string]any{"type": "string", "description": "Node (optional for list)"},
			"task_id":   map[string]any{"type": "string", "description": "Task UPID (status, log, cancel)"},
			"start":     map[string]any{"type": "integer", "description": "Log start line (optional)"},
			"limit":     map[string]any{"type": "integer", "description": "Log line count (optional)"},
		}, CategoryDefault},
		{"manage_pools", "Pool outcome: list, get, create, update, delete, members. Confirm required for delete.", s.managePools, map[string]any{
			"action":  map[string]any{"type": "string", "description": "list, get, create, update, delete, members"},
			"poolid":  map[string]any{"type": "string", "description": "Pool ID (get, create, update, delete, members)"},
			"comment": map[string]any{"type": "string", "description": "Comment (create/update, optional)"},
			"members": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Members (create/update, optional)"},
			"delete":  map[string]any{"type": "boolean", "description": "Update: remove members (optional)"},
			"confirm": map[string]any{"type": "string", "description": "For delete: set to poolid"},
		}, CategoryDefault},
		{"manage_node", "Node outcome: update_config, apply_updates, reboot, or shutdown", s.manageNode, map[string]any{
			"action":    map[string]any{"type": "string", "description": "update_config, apply_updates, reboot, shutdown"},
			"node_name": map[string]any{"type": "string", "description": "Node name"},
		}, CategoryDefault},
		{"list_access", "List users, groups, roles, or ACL. Optionally pass userid to get one user.", s.listAccess, map[string]any{
			"what":   map[string]any{"type": "string", "description": "users, groups, roles, or acl"},
			"userid": map[string]any{"type": "string", "description": "Optional: get single user details"},
		}, CategoryDefault},
		{"manage_access", "Access outcome: create/update/delete user, change_password, create/delete group, create/delete role, set_acl, create/delete token. Use action; confirm for deletes.", s.manageAccess, map[string]any{
			"action":    map[string]any{"type": "string", "description": "create_user, update_user, delete_user, change_password, create_group, delete_group, create_role, delete_role, set_acl, create_token, delete_token"},
			"userid":    map[string]any{"type": "string", "description": "User ID (many actions)"},
			"password":  map[string]any{"type": "string", "description": "Password (create_user, change_password)"},
			"email":     map[string]any{"type": "string", "description": "Email (optional)"},
			"comment":   map[string]any{"type": "string", "description": "Comment (optional)"},
			"groupid":   map[string]any{"type": "string", "description": "Group ID"},
			"roleid":    map[string]any{"type": "string", "description": "Role ID"},
			"privs":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Privileges (create_role)"},
			"path":      map[string]any{"type": "string", "description": "ACL path (set_acl)"},
			"role":      map[string]any{"type": "string", "description": "Role (set_acl)"},
			"tokenid":   map[string]any{"type": "string", "description": "Token ID"},
			"expire":    map[string]any{"type": "integer", "description": "Expiry (create_token, optional)"},
			"propagate": map[string]any{"type": "boolean", "description": "Propagate (set_acl, optional)"},
			"confirm":   map[string]any{"type": "string", "description": "For delete_*: set to resource id or userid/tokenid"},
		}, CategoryDefault},
		{"manage_firewall", "Firewall outcome: list cluster rules, list guest rules, create rule, delete rule, list groups, create group. Confirm for delete rule.", s.manageFirewall, map[string]any{
			"action":     map[string]any{"type": "string", "description": "list_cluster, list_guest, create_rule, delete_rule, list_groups, create_group"},
			"node_name": map[string]any{"type": "string", "description": "Node (list_guest)"},
			"vmid":      map[string]any{"type": "integer", "description": "VM ID (list_guest)"},
			"position":  map[string]any{"type": "string", "description": "Rule position (delete_rule)"},
			"direction": map[string]any{"type": "string", "description": "in, out, group (create_rule)"},
			"action_rule": map[string]any{"type": "string", "description": "ACCEPT, DROP, REJECT (create_rule)"},
			"source":   map[string]any{"type": "string", "description": "Source (optional)"},
			"dest":     map[string]any{"type": "string", "description": "Dest (optional)"},
			"proto":    map[string]any{"type": "string", "description": "Protocol (optional)"},
			"sport":    map[string]any{"type": "string", "description": "Source port (optional)"},
			"dport":    map[string]any{"type": "string", "description": "Dest port (optional)"},
			"comment":  map[string]any{"type": "string", "description": "Comment (optional)"},
			"enable":   map[string]any{"type": "integer", "description": "0 or 1 (create_rule, optional)"},
			"name":     map[string]any{"type": "string", "description": "Group name (create_group)"},
			"confirm":  map[string]any{"type": "string", "description": "For delete_rule: set to position"},
		}, CategoryDefault},
		{"get_ha_status", "Get cluster HA status", s.getHAStatus, map[string]any{}, CategoryDefault},
		{"manage_ha", "HA outcome: enable or disable HA for a resource", s.manageHA, map[string]any{
			"action":  map[string]any{"type": "string", "description": "enable or disable"},
			"sid":     map[string]any{"type": "string", "description": "Resource ID (e.g. vm:100)"},
			"comment": map[string]any{"type": "string", "description": "Comment (enable, optional)"},
			"state":   map[string]any{"type": "string", "description": "started or stopped (enable, optional)"},
		}, CategoryDefault},
		{"manage_cluster", "Cluster outcome: config, nodes_status, add_node, or remove_node", s.manageCluster, map[string]any{
			"action":          map[string]any{"type": "string", "description": "config, nodes_status, add_node, remove_node"},
			"node_name":       map[string]any{"type": "string", "description": "Node (add_node, remove_node)"},
			"cluster_name":    map[string]any{"type": "string", "description": "Cluster name (add_node, optional)"},
			"cluster_network": map[string]any{"type": "string", "description": "Cluster network (add_node, optional)"},
		}, CategoryDefault},
	}
}

// getCluster returns cluster overview (nodes, resources, status).
func (s *Server) getCluster(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return s.getClusterOverview(ctx, request)
}

// getNode returns one node's details by detail type (status, config, disks, cert, logs, network, dns, stats, apt).
func (s *Server) getNode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node")
	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name is required"), nil
	}
	detail := strings.ToLower(request.GetString("detail", "status"))
	var out interface{}
	var err error
	switch detail {
	case "status":
		out, err = s.proxmoxClient.GetNode(ctx, nodeName)
	case "config":
		out, err = s.proxmoxClient.GetNodeConfig(ctx, nodeName)
	case "disks":
		out, err = s.proxmoxClient.GetNodeDisks(ctx, nodeName)
	case "cert":
		out, err = s.proxmoxClient.GetNodeCert(ctx, nodeName)
	case "logs":
		out, err = s.proxmoxClient.GetNodeLogs(ctx, nodeName, request.GetInt("lines", 50))
	case "network":
		out, err = s.proxmoxClient.GetNodeNetwork(ctx, nodeName)
	case "dns":
		out, err = s.proxmoxClient.GetNodeDNS(ctx, nodeName)
	case "stats":
		out, err = s.proxmoxClient.GetNodeStats(ctx, nodeName, "")
	case "apt":
		out, err = s.proxmoxClient.GetNodeAPTUpdates(ctx, nodeName)
	default:
		return mcp.NewToolResultError("detail must be one of: status, config, disks, cert, logs, network, dns, stats, apt"), nil
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get_node failed: %v", err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{"node": nodeName, "detail": detail, "data": out})
}

// manageStorage dispatches by action: list, info, content, create, update, delete, quota.
func (s *Server) manageStorage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: manage_storage")
	action := strings.ToLower(request.GetString("action", ""))
	nodeName := request.GetString("node_name", "")
	storage := request.GetString("storage", "")

	switch action {
	case "list":
		if nodeName != "" {
			st, err := s.proxmoxClient.GetNodeStorage(ctx, nodeName)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list storage failed: %v", err)), nil
			}
			return mcp.NewToolResultJSON(map[string]interface{}{"node": nodeName, "storage": st, "count": len(st)})
		}
		st, err := s.proxmoxClient.GetStorage(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list storage failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"storage": st, "count": len(st)})
	case "info":
		if storage == "" {
			return mcp.NewToolResultError("storage is required for action info"), nil
		}
		out, err := s.proxmoxClient.GetStorageInfo(ctx, storage)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("storage info failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"storage": storage, "info": out})
	case "content":
		if storage == "" {
			return mcp.NewToolResultError("storage is required for action content"), nil
		}
		out, err := s.proxmoxClient.GetStorageContent(ctx, storage)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("storage content failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"storage": storage, "content": out})
	case "create":
		if storage == "" {
			return mcp.NewToolResultError("storage and storage_type and content are required for create"), nil
		}
		storageType := request.GetString("storage_type", "")
		content := request.GetString("content", "")
		if storageType == "" || content == "" {
			return mcp.NewToolResultError("storage_type and content are required for create"), nil
		}
		config := make(map[string]interface{})
		if v := request.GetArguments()["config"]; v != nil {
			jsonBytes, _ := json.Marshal(v)
			_ = json.Unmarshal(jsonBytes, &config)
		}
		result, err := s.proxmoxClient.CreateStorage(ctx, storage, storageType, content, config)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create storage failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "create", "storage": storage, "result": result})
	case "update":
		if storage == "" {
			return mcp.NewToolResultError("storage is required for update"), nil
		}
		config := make(map[string]interface{})
		if v := request.GetArguments()["config"]; v != nil {
			jsonBytes, _ := json.Marshal(v)
			_ = json.Unmarshal(jsonBytes, &config)
		}
		result, err := s.proxmoxClient.UpdateStorage(ctx, storage, config)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update storage failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "update", "storage": storage, "result": result})
	case "delete":
		if storage == "" {
			return mcp.NewToolResultError("storage is required for delete"), nil
		}
		if request.GetString("confirm", "") != storage {
			return mcp.NewToolResultError(fmt.Sprintf("delete requires confirm set to %q", storage)), nil
		}
		result, err := s.proxmoxClient.DeleteStorage(ctx, storage)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete storage failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "delete", "storage": storage, "result": result})
	case "quota":
		if storage == "" {
			return mcp.NewToolResultError("storage is required for quota"), nil
		}
		out, err := s.proxmoxClient.GetStorageQuota(ctx, storage)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("storage quota failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"storage": storage, "quota": out})
	default:
		return mcp.NewToolResultError("action must be one of: list, info, content, create, update, delete, quota"), nil
	}
}

// getGuest returns one guest's detail (status, config, console, firewall, stats).
func (s *Server) getGuest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_guest")
	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name is required"), nil
	}
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	if guestType != "vm" && guestType != "lxc" {
		return mcp.NewToolResultError("guest_type must be vm or lxc"), nil
	}
	vmid := request.GetInt("vmid", 0)
	if vmid <= 0 {
		return mcp.NewToolResultError("vmid is required"), nil
	}
	detail := strings.ToLower(request.GetString("detail", "status"))

	var out interface{}
	var err error
	if guestType == "vm" {
		switch detail {
		case "status":
			out, err = s.proxmoxClient.GetVM(ctx, nodeName, vmid)
		case "config":
			out, err = s.proxmoxClient.GetVMConfig(ctx, nodeName, vmid)
		case "console":
			out, err = s.proxmoxClient.GetVMConsole(ctx, nodeName, vmid)
		case "firewall":
			out, err = s.proxmoxClient.GetVMFirewallRules(ctx, nodeName, vmid)
		case "stats":
			out, err = s.proxmoxClient.GetVMStats(ctx, nodeName, vmid)
		default:
			return mcp.NewToolResultError("detail must be one of: status, config, console, firewall, stats"), nil
		}
	} else {
		switch detail {
		case "status":
			out, err = s.proxmoxClient.GetContainer(ctx, nodeName, vmid)
		case "config":
			out, err = s.proxmoxClient.GetContainerConfig(ctx, nodeName, vmid)
		case "console":
			return mcp.NewToolResultError("console detail is for VMs only"), nil
		case "firewall":
			return mcp.NewToolResultError("firewall detail is for VMs only"), nil
		case "stats":
			out, err = s.proxmoxClient.GetContainerStats(ctx, nodeName, vmid)
		default:
			return mcp.NewToolResultError("detail must be one of: status, config, stats"), nil
		}
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get_guest failed: %v", err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{"node": nodeName, "guest_type": guestType, "vmid": vmid, "detail": detail, "data": out})
}

// createGuest creates a VM or container (dispatches to createVM or createContainer).
func (s *Server) createGuest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	if guestType == "vm" {
		return s.createVM(ctx, request)
	}
	// createContainer accepts vmid when container_id is not set (see server.go createContainer).
	return s.createContainer(ctx, request)
}

// deleteGuest deletes a VM or container; requires confirm.
func (s *Server) deleteGuest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	nodeName := request.GetString("node_name", "")
	vmid := request.GetInt("vmid", 0)
	if nodeName == "" || vmid <= 0 {
		return mcp.NewToolResultError("guest_type, node_name, and vmid are required"), nil
	}
	expectedConfirm := fmt.Sprintf("%s/%d", nodeName, vmid)
	if request.GetString("confirm", "") != expectedConfirm {
		return mcp.NewToolResultError(fmt.Sprintf("delete requires confirm set to %q", expectedConfirm)), nil
	}
	force := request.GetBool("force", false)
	if guestType == "vm" {
		result, err := s.proxmoxClient.DeleteVM(ctx, nodeName, vmid, force)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete VM failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "delete", "guest_type": "vm", "vmid": vmid, "node": nodeName, "result": result})
	}
	result, err := s.proxmoxClient.DeleteContainer(ctx, nodeName, vmid, force)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("delete container failed: %v", err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{"action": "delete", "guest_type": "lxc", "vmid": vmid, "node": nodeName, "result": result})
}

// manageSnapshot: action create, list, delete, restore.
func (s *Server) manageSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := strings.ToLower(request.GetString("action", ""))
	switch action {
	case "create":
		return s.createSnapshot(ctx, request)
	case "list":
		return s.listSnapshots(ctx, request)
	case "delete":
		return s.deleteSnapshot(ctx, request)
	case "restore":
		return s.restoreSnapshot(ctx, request)
	default:
		return mcp.NewToolResultError("action must be one of: create, list, delete, restore"), nil
	}
}

// manageBackup: action create, restore, delete.
func (s *Server) manageBackup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := strings.ToLower(request.GetString("action", ""))
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	nodeName := request.GetString("node_name", "")
	vmid := request.GetInt("vmid", 0)
	backupID := request.GetString("backup_id", "")
	storage := request.GetString("storage", "")

	switch action {
	case "create":
		if nodeName == "" || vmid <= 0 || storage == "" {
			return mcp.NewToolResultError("node_name, vmid, and storage are required for create"), nil
		}
		notes := request.GetString("notes", "")
		if guestType == "lxc" {
			result, err := s.proxmoxClient.CreateContainerBackup(ctx, nodeName, vmid, storage, backupID, notes)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("create backup failed: %v", err)), nil
			}
			return mcp.NewToolResultJSON(map[string]interface{}{"action": "create", "guest_type": "lxc", "vmid": vmid, "result": result})
		}
		result, err := s.proxmoxClient.CreateVMBackup(ctx, nodeName, vmid, storage, backupID, notes)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create backup failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "create", "guest_type": "vm", "vmid": vmid, "result": result})
	case "restore":
		if nodeName == "" || backupID == "" || storage == "" {
			return mcp.NewToolResultError("node_name, backup_id, and storage are required for restore"), nil
		}
		if guestType == "lxc" {
			result, err := s.proxmoxClient.RestoreContainerBackup(ctx, nodeName, backupID, storage)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("restore failed: %v", err)), nil
			}
			return mcp.NewToolResultJSON(map[string]interface{}{"action": "restore", "guest_type": "lxc", "result": result})
		}
		result, err := s.proxmoxClient.RestoreVMBackup(ctx, nodeName, backupID, storage)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("restore failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "restore", "guest_type": "vm", "result": result})
	case "delete":
		if storage == "" || backupID == "" {
			return mcp.NewToolResultError("storage and backup_id are required for delete"), nil
		}
		expectedConfirm := fmt.Sprintf("%s/%s", storage, backupID)
		if request.GetString("confirm", "") != expectedConfirm {
			return mcp.NewToolResultError(fmt.Sprintf("delete requires confirm set to %q", expectedConfirm)), nil
		}
		result, err := s.proxmoxClient.DeleteBackup(ctx, storage, backupID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete backup failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "delete", "result": result})
	default:
		return mcp.NewToolResultError("action must be one of: create, restore, delete"), nil
	}
}

// manageTasks: action list, status, log, cancel.
func (s *Server) manageTasks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := strings.ToLower(request.GetString("action", ""))
	nodeName := request.GetString("node_name", "")
	taskID := request.GetString("task_id", "")

	switch action {
	case "list":
		if nodeName != "" {
			tasks, err := s.proxmoxClient.GetNodeTasks(ctx, nodeName)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list tasks failed: %v", err)), nil
			}
			return mcp.NewToolResultJSON(map[string]interface{}{"node": nodeName, "tasks": tasks, "count": len(tasks)})
		}
		tasks, err := s.proxmoxClient.GetClusterTasks(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list tasks failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"tasks": tasks, "count": len(tasks)})
	case "status":
		if taskID == "" {
			return mcp.NewToolResultError("task_id is required for status"), nil
		}
		out, err := s.proxmoxClient.GetTaskStatus(ctx, taskID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("task status failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(out)
	case "log":
		if taskID == "" {
			return mcp.NewToolResultError("task_id is required for log"), nil
		}
		out, err := s.proxmoxClient.GetTaskLog(ctx, taskID, request.GetInt("start", 0), request.GetInt("limit", 100))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("task log failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(out)
	case "cancel":
		if taskID == "" {
			return mcp.NewToolResultError("task_id is required for cancel"), nil
		}
		result, err := s.proxmoxClient.CancelTask(ctx, taskID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("cancel task failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "cancel", "task_id": taskID, "result": result})
	default:
		return mcp.NewToolResultError("action must be one of: list, status, log, cancel"), nil
	}
}

// managePools: action list, get, create, update, delete, members.
func (s *Server) managePools(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := strings.ToLower(request.GetString("action", ""))
	poolid := request.GetString("poolid", "")

	switch action {
	case "list":
		pools, err := s.proxmoxClient.ListPools(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list pools failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"pools": pools, "count": len(pools)})
	case "get":
		if poolid == "" {
			return mcp.NewToolResultError("poolid is required for get"), nil
		}
		out, err := s.proxmoxClient.GetPool(ctx, poolid)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get pool failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(out)
	case "create":
		if poolid == "" {
			return mcp.NewToolResultError("poolid is required for create"), nil
		}
		comment := request.GetString("comment", "")
		var members []string
		if v := request.GetArguments()["members"]; v != nil {
			jsonBytes, _ := json.Marshal(v)
			_ = json.Unmarshal(jsonBytes, &members)
		}
		result, err := s.proxmoxClient.CreatePool(ctx, poolid, comment, members)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create pool failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "create", "poolid": poolid, "result": result})
	case "update":
		if poolid == "" {
			return mcp.NewToolResultError("poolid is required for update"), nil
		}
		comment := request.GetString("comment", "")
		var members []string
		if v := request.GetArguments()["members"]; v != nil {
			jsonBytes, _ := json.Marshal(v)
			_ = json.Unmarshal(jsonBytes, &members)
		}
		del := request.GetBool("delete", false)
		result, err := s.proxmoxClient.UpdatePool(ctx, poolid, comment, members, del)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update pool failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "update", "poolid": poolid, "result": result})
	case "delete":
		if poolid == "" {
			return mcp.NewToolResultError("poolid is required for delete"), nil
		}
		if request.GetString("confirm", "") != poolid {
			return mcp.NewToolResultError(fmt.Sprintf("delete requires confirm set to %q", poolid)), nil
		}
		result, err := s.proxmoxClient.DeletePool(ctx, poolid)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete pool failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "delete", "poolid": poolid, "result": result})
	case "members":
		if poolid == "" {
			return mcp.NewToolResultError("poolid is required for members"), nil
		}
		out, err := s.proxmoxClient.GetPoolMembers(ctx, poolid)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get pool members failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"poolid": poolid, "members": out})
	default:
		return mcp.NewToolResultError("action must be one of: list, get, create, update, delete, members"), nil
	}
}

// manageNode: action update_config, apply_updates, reboot, shutdown.
func (s *Server) manageNode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := strings.ToLower(request.GetString("action", ""))
	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name is required"), nil
	}
	switch action {
	case "update_config":
		config := make(map[string]interface{})
		if v := request.GetArguments()["config"]; v != nil {
			jsonBytes, _ := json.Marshal(v)
			_ = json.Unmarshal(jsonBytes, &config)
		}
		result, err := s.proxmoxClient.UpdateNodeConfig(ctx, nodeName, config)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update node config failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "update_config", "node": nodeName, "result": result})
	case "apply_updates":
		result, err := s.proxmoxClient.ApplyNodeUpdates(ctx, nodeName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("apply updates failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "apply_updates", "node": nodeName, "result": result})
	case "reboot":
		result, err := s.proxmoxClient.RebootNode(ctx, nodeName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("reboot failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "reboot", "node": nodeName, "result": result})
	case "shutdown":
		result, err := s.proxmoxClient.ShutdownNode(ctx, nodeName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("shutdown failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "shutdown", "node": nodeName, "result": result})
	default:
		return mcp.NewToolResultError("action must be one of: update_config, apply_updates, reboot, shutdown"), nil
	}
}

// listAccess: what = users, groups, roles, acl. Optional userid for single user.
func (s *Server) listAccess(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	what := strings.ToLower(request.GetString("what", "users"))
	userid := request.GetString("userid", "")
	if userid != "" {
		user, err := s.proxmoxClient.GetUser(ctx, userid)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get user failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"user": user})
	}
	switch what {
	case "users":
		users, err := s.proxmoxClient.ListUsers(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list users failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"users": users, "count": len(users)})
	case "groups":
		groups, err := s.proxmoxClient.ListGroups(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list groups failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"groups": groups, "count": len(groups)})
	case "roles":
		roles, err := s.proxmoxClient.ListRoles(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list roles failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"roles": roles, "count": len(roles)})
	case "acl":
		acls, err := s.proxmoxClient.ListACLs(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list acl failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"acl": acls, "count": len(acls)})
	default:
		return mcp.NewToolResultError("what must be one of: users, groups, roles, acl"), nil
	}
}

// manageAccess: single tool for all access operations (create_user, update_user, delete_user, etc.).
func (s *Server) manageAccess(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := strings.ToLower(request.GetString("action", ""))
	switch action {
	case "create_user":
		return s.createUser(ctx, request)
	case "update_user":
		return s.updateUser(ctx, request)
	case "delete_user":
		return s.deleteUser(ctx, request)
	case "change_password":
		return s.changePassword(ctx, request)
	case "create_group":
		return s.createGroup(ctx, request)
	case "delete_group":
		return s.deleteGroup(ctx, request)
	case "create_role":
		return s.createRole(ctx, request)
	case "delete_role":
		return s.deleteRole(ctx, request)
	case "set_acl":
		return s.setACL(ctx, request)
	case "create_token":
		return s.createAPIToken(ctx, request)
	case "delete_token":
		return s.deleteAPIToken(ctx, request)
	default:
		return mcp.NewToolResultError("action must be one of: create_user, update_user, delete_user, change_password, create_group, delete_group, create_role, delete_role, set_acl, create_token, delete_token"), nil
	}
}

// manageFirewall: list_cluster, list_guest, create_rule, delete_rule, list_groups, create_group.
func (s *Server) manageFirewall(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := strings.ToLower(request.GetString("action", ""))
	switch action {
	case "list_cluster":
		rules, err := s.proxmoxClient.GetFirewallRules(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list firewall failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"rules": rules, "count": len(rules)})
	case "list_guest":
		nodeName := request.GetString("node_name", "")
		vmid := request.GetInt("vmid", 0)
		if nodeName == "" || vmid <= 0 {
			return mcp.NewToolResultError("node_name and vmid are required for list_guest"), nil
		}
		rules, err := s.proxmoxClient.GetVMFirewallRules(ctx, nodeName, vmid)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list guest firewall failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"node": nodeName, "vmid": vmid, "rules": rules})
	case "create_rule":
		return s.createFirewallRule(ctx, request)
	case "delete_rule":
		return s.deleteFirewallRule(ctx, request)
	case "list_groups":
		groups, err := s.proxmoxClient.GetSecurityGroups(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list security groups failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"groups": groups, "count": len(groups)})
	case "create_group":
		return s.createSecurityGroup(ctx, request)
	default:
		return mcp.NewToolResultError("action must be one of: list_cluster, list_guest, create_rule, delete_rule, list_groups, create_group"), nil
	}
}

// manageHA: action enable or disable.
func (s *Server) manageHA(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := strings.ToLower(request.GetString("action", ""))
	sid := request.GetString("sid", "")
	if sid == "" {
		return mcp.NewToolResultError("sid is required"), nil
	}
	if action == "enable" {
		comment := request.GetString("comment", "")
		state := request.GetString("state", "")
		result, err := s.proxmoxClient.EnableHAResource(ctx, sid, comment, state)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("enable HA failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "enable", "sid": sid, "result": result})
	}
	if action == "disable" {
		result, err := s.proxmoxClient.DisableHAResource(ctx, sid)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("disable HA failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "disable", "sid": sid, "result": result})
	}
	return mcp.NewToolResultError("action must be enable or disable"), nil
}

// manageCluster: action config, nodes_status, add_node, remove_node.
func (s *Server) manageCluster(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := strings.ToLower(request.GetString("action", ""))
	switch action {
	case "config":
		out, err := s.proxmoxClient.GetClusterConfig(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get cluster config failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(out)
	case "nodes_status":
		out, err := s.proxmoxClient.GetClusterNodesStatus(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get cluster nodes status failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(out)
	case "add_node":
		nodeName := request.GetString("node_name", "")
		if nodeName == "" {
			return mcp.NewToolResultError("node_name is required for add_node"), nil
		}
		clusterName := request.GetString("cluster_name", "")
		clusterNetwork := request.GetString("cluster_network", "")
		result, err := s.proxmoxClient.AddNodeToCluster(ctx, nodeName, clusterName, clusterNetwork)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("add node failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "add_node", "node": nodeName, "result": result})
	case "remove_node":
		nodeName := request.GetString("node_name", "")
		if nodeName == "" {
			return mcp.NewToolResultError("node_name is required for remove_node"), nil
		}
		result, err := s.proxmoxClient.RemoveNodeFromCluster(ctx, nodeName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("remove node failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{"action": "remove_node", "node": nodeName, "result": result})
	default:
		return mcp.NewToolResultError("action must be one of: config, nodes_status, add_node, remove_node"), nil
	}
}
