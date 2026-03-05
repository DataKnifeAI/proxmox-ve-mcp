package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
	"github.com/surrealwolf/proxmox-ve-mcp/internal/proxmox"
)

// Server represents the MCP server
type Server struct {
	proxmoxClient *proxmox.Client
	server        *server.MCPServer
	logger        *logrus.Entry
}

// NewServer creates a new MCP server
func NewServer(proxmoxClient *proxmox.Client) *Server {
	s := &Server{
		proxmoxClient: proxmoxClient,
		server:        server.NewMCPServer("proxmox-ve-mcp", "0.1.0"),
		logger:        logrus.WithField("component", "MCPServer"),
	}

	s.registerTools()
	return s
}

// getArrayLength returns the length of an array if the value is an array, otherwise returns 0
func getArrayLength(v interface{}) int {
	if arr, ok := v.([]interface{}); ok {
		return len(arr)
	}
	// Try to unmarshal if it's raw JSON
	if data, err := json.Marshal(v); err == nil {
		var arr []interface{}
		if json.Unmarshal(data, &arr) == nil {
			return len(arr)
		}
	}
	return 0
}

// ToolCategory represents the category/priority of a tool
type ToolCategory string

const (
	CategoryDefault  ToolCategory = "default"  // Common tools enabled by default
	CategoryAdvanced ToolCategory = "advanced" // Advanced tools require explicit enablement
)

// ToolDefinition holds tool metadata for lazy loading
type ToolDefinition struct {
	Name        string
	Description string
	Handler     server.ToolHandlerFunc
	Properties  map[string]any
	Category    ToolCategory
}

func (s *Server) registerTools() {
	toolDefs := s.outcomeBasedToolDefs()
	for _, def := range toolDefs {
		s.server.AddTool(mcp.Tool{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: mcp.ToolInputSchema{Type: "object", Properties: def.Properties},
		}, def.Handler)
	}
	s.logger.Infof("Registered %d outcome-based tools", len(toolDefs))
}

// ServeStdio starts the MCP server with stdio transport
func (s *Server) ServeStdio(ctx context.Context) error {
	s.logger.Info("Starting Proxmox VE MCP Server")
	return server.ServeStdio(s.server)
}

// ServeHTTP starts the MCP server with HTTP transport
func (s *Server) ServeHTTP(addr string, ctx context.Context) error {
	s.logger.Infof("Starting Proxmox VE MCP Server on HTTP at %s", addr)

	http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Parse the MCP request
		var requestData map[string]interface{}
		if err := json.Unmarshal(body, &requestData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Log the request
		s.logger.Debugf("HTTP MCP request received: %v", requestData)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{
			"status": "MCP HTTP transport is available",
			"info":   "This is an HTTP endpoint. Use stdio transport for full MCP protocol support.",
		}
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	return http.ListenAndServe(addr, nil)
}

// getNodes handles the get_nodes tool
func (s *Server) getNodes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_nodes")

	nodes, err := s.proxmoxClient.GetNodes(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get nodes: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// getNodeStatus handles the get_node_status tool
func (s *Server) getNodeStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_status")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	node, err := s.proxmoxClient.GetNode(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node status: %v", err)), nil
	}

	return mcp.NewToolResultJSON(node)
}

// getVMs handles the get_vms tool
func (s *Server) getVMs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_vms")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vms, err := s.proxmoxClient.GetVMs(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get VMs: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"vms":   vms,
		"count": len(vms),
		"node":  nodeName,
	})
}

// getVMStatus handles the get_vm_status tool
func (s *Server) getVMStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_vm_status")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	vm, err := s.proxmoxClient.GetVM(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get VM status: %v", err)), nil
	}

	return mcp.NewToolResultJSON(vm)
}

// getContainers handles the get_containers tool
func (s *Server) getContainers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_containers")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containers, err := s.proxmoxClient.GetContainers(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get containers: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"containers": containers,
		"count":      len(containers),
		"node":       nodeName,
	})
}

// getContainerStatus handles the get_container_status tool
func (s *Server) getContainerStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_container_status")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	container, err := s.proxmoxClient.GetContainer(ctx, nodeName, containerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get container status: %v", err)), nil
	}

	return mcp.NewToolResultJSON(container)
}

// getClusterResources handles the get_cluster_resources tool
func (s *Server) getClusterResources(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_cluster_resources")

	resources, err := s.proxmoxClient.GetClusterResources(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get cluster resources: %v", err)), nil
	}

	// Wrap array response in object for MCP compatibility
	return mcp.NewToolResultJSON(map[string]interface{}{
		"resources": resources,
		"count":     getArrayLength(resources),
	})
}

// getClusterStatus handles the get_cluster_status tool
func (s *Server) getClusterStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_cluster_status")

	status, err := s.proxmoxClient.GetClusterStatus(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get cluster status: %v", err)), nil
	}

	// Wrap array response in object for MCP compatibility
	return mcp.NewToolResultJSON(map[string]interface{}{
		"status": status,
		"count":  getArrayLength(status),
	})
}

// getClusterOverview returns nodes, resources, and status in one call (outcome-based replacement for get_nodes + get_cluster_resources + get_cluster_status)
func (s *Server) getClusterOverview(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_cluster_overview")

	nodes, err := s.proxmoxClient.GetNodes(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get nodes: %v", err)), nil
	}
	resources, err := s.proxmoxClient.GetClusterResources(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get cluster resources: %v", err)), nil
	}
	status, err := s.proxmoxClient.GetClusterStatus(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get cluster status: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"nodes":     nodes,
		"nodes_count": len(nodes),
		"resources": resources,
		"resources_count": getArrayLength(resources),
		"status":    status,
		"status_count": getArrayLength(status),
	})
}

// listGuests returns VMs and/or LXC containers (outcome-based replacement for get_vms + get_containers)
func (s *Server) listGuests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_guests")

	nodeName := request.GetString("node_name", "")
	guestType := strings.ToLower(request.GetString("type", "all"))
	if guestType != "vm" && guestType != "lxc" && guestType != "all" {
		return mcp.NewToolResultError("type must be vm, lxc, or all"), nil
	}

	// If no node specified, use cluster resources and filter
	if nodeName == "" {
		resources, err := s.proxmoxClient.GetClusterResources(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get cluster resources: %v", err)), nil
		}
		resourceList, ok := resources.([]interface{})
		if !ok {
			return mcp.NewToolResultError("Unexpected cluster resources format"), nil
		}
		var vms, containers []interface{}
		for _, r := range resourceList {
			m, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			t, _ := m["type"].(string)
			if guestType == "all" || guestType == "vm" {
				if t == "qemu" {
					vms = append(vms, r)
				}
			}
			if guestType == "all" || guestType == "lxc" {
				if t == "lxc" {
					containers = append(containers, r)
				}
			}
		}
		return mcp.NewToolResultJSON(map[string]interface{}{
			"vms":        vms,
			"vms_count":  len(vms),
			"containers": containers,
			"containers_count": len(containers),
			"type":       guestType,
		})
	}

	out := map[string]interface{}{"node": nodeName, "type": guestType}
	if guestType == "all" || guestType == "vm" {
		vms, err := s.proxmoxClient.GetVMs(ctx, nodeName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get VMs: %v", err)), nil
		}
		out["vms"] = vms
		out["vms_count"] = len(vms)
	}
	if guestType == "all" || guestType == "lxc" {
		containers, err := s.proxmoxClient.GetContainers(ctx, nodeName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get containers: %v", err)), nil
		}
		out["containers"] = containers
		out["containers_count"] = len(containers)
	}
	return mcp.NewToolResultJSON(out)
}

// controlGuest starts, stops, shuts down, or reboots a VM or container (outcome-based replacement for start/stop/shutdown/reboot per type)
func (s *Server) controlGuest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: control_guest")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	if guestType != "vm" && guestType != "lxc" {
		return mcp.NewToolResultError("guest_type must be vm or lxc"), nil
	}
	vmid := request.GetInt("vmid", 0)
	if vmid <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}
	action := strings.ToLower(request.GetString("action", ""))
	if action != "start" && action != "stop" && action != "shutdown" && action != "reboot" && action != "suspend" && action != "resume" && action != "migrate" {
		return mcp.NewToolResultError("action must be start, stop, shutdown, reboot, suspend, resume, or migrate"), nil
	}
	if guestType == "lxc" && (action == "suspend" || action == "resume" || action == "migrate") {
		return mcp.NewToolResultError("suspend, resume, and migrate are only supported for VMs"), nil
	}
	if action == "migrate" {
		targetNode := request.GetString("target_node", "")
		if targetNode == "" {
			return mcp.NewToolResultError("target_node is required for action migrate"), nil
		}
		online := request.GetBool("online", false)
		result, err := s.proxmoxClient.MigrateVM(ctx, nodeName, vmid, targetNode, online)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("migrate failed: %v", err)), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{
			"action": "migrate", "guest_type": guestType, "vmid": vmid, "node": nodeName, "target_node": targetNode, "result": result,
		})
	}

	var result interface{}
	var err error
	if guestType == "vm" {
		switch action {
		case "start":
			result, err = s.proxmoxClient.StartVM(ctx, nodeName, vmid)
		case "stop":
			result, err = s.proxmoxClient.StopVM(ctx, nodeName, vmid)
		case "shutdown":
			result, err = s.proxmoxClient.ShutdownVM(ctx, nodeName, vmid)
		case "reboot":
			result, err = s.proxmoxClient.RebootVM(ctx, nodeName, vmid)
		case "suspend":
			result, err = s.proxmoxClient.SuspendVM(ctx, nodeName, vmid)
		case "resume":
			result, err = s.proxmoxClient.ResumeVM(ctx, nodeName, vmid)
		}
	} else {
		switch action {
		case "start":
			result, err = s.proxmoxClient.StartContainer(ctx, nodeName, vmid)
		case "stop":
			result, err = s.proxmoxClient.StopContainer(ctx, nodeName, vmid)
		case "shutdown":
			result, err = s.proxmoxClient.ShutdownContainer(ctx, nodeName, vmid)
		case "reboot":
			result, err = s.proxmoxClient.RebootContainer(ctx, nodeName, vmid)
		}
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to %s %s: %v", action, guestType, err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":     action,
		"guest_type": guestType,
		"vmid":       vmid,
		"node":       nodeName,
		"result":    result,
	})
}

// getStorage handles the get_storage tool
func (s *Server) getStorage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_storage")

	storage, err := s.proxmoxClient.GetStorage(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get storage: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"storage": storage,
		"count":   len(storage),
	})
}

// getNodeStorage handles the get_node_storage tool
func (s *Server) getNodeStorage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_storage")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	storage, err := s.proxmoxClient.GetNodeStorage(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node storage: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"storage": storage,
		"node":    nodeName,
		"count":   len(storage),
	})
}

// startVM handles the start_vm tool
func (s *Server) startVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: start_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.StartVM(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to start VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "start",
		"vmid":   vmID,
		"node":   nodeName,
		"result": result,
	})
}

// stopVM handles the stop_vm tool
func (s *Server) stopVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: stop_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.StopVM(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to stop VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "stop",
		"vmid":   vmID,
		"node":   nodeName,
		"result": result,
	})
}

// shutdownVM handles the shutdown_vm tool
func (s *Server) shutdownVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: shutdown_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.ShutdownVM(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to shutdown VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "shutdown",
		"vmid":   vmID,
		"node":   nodeName,
		"result": result,
	})
}

// rebootVM handles the reboot_vm tool
func (s *Server) rebootVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: reboot_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.RebootVM(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to reboot VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "reboot",
		"vmid":   vmID,
		"node":   nodeName,
		"result": result,
	})
}

// getVMConfig handles the get_vm_config tool
func (s *Server) getVMConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_vm_config")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	config, err := s.proxmoxClient.GetVMConfig(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get VM config: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"vmid":   vmID,
		"node":   nodeName,
		"config": config,
	})
}

// deleteVM handles the delete_vm tool
func (s *Server) deleteVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	expectedConfirm := fmt.Sprintf("%s/%d", nodeName, vmID)
	confirm := request.GetString("confirm", "")
	if confirm != expectedConfirm {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", expectedConfirm)), nil
	}

	force := request.GetBool("force", false)

	result, err := s.proxmoxClient.DeleteVM(ctx, nodeName, vmID, force)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "delete",
		"vmid":   vmID,
		"node":   nodeName,
		"force":  force,
		"result": result,
	})
}

// suspendVM handles the suspend_vm tool
func (s *Server) suspendVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: suspend_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.SuspendVM(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to suspend VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "suspend",
		"vmid":   vmID,
		"node":   nodeName,
		"result": result,
	})
}

// resumeVM handles the resume_vm tool
func (s *Server) resumeVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: resume_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.ResumeVM(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to resume VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "resume",
		"vmid":   vmID,
		"node":   nodeName,
		"result": result,
	})
}

// createVM handles the create_vm tool (single tool with all optional flat params)
func (s *Server) createVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	config := map[string]interface{}{"vmid": vmID}
	if name := request.GetString("name", ""); name != "" {
		config["name"] = name
	}
	if memory := request.GetInt("memory", 0); memory > 0 {
		config["memory"] = memory
	}
	if cores := request.GetInt("cores", 0); cores > 0 {
		config["cores"] = cores
	}
	if sockets := request.GetInt("sockets", 0); sockets > 0 {
		config["sockets"] = sockets
	}
	if ide2 := request.GetString("ide2", ""); ide2 != "" {
		config["ide2"] = ide2
	}
	if sata0 := request.GetString("sata0", ""); sata0 != "" {
		config["sata0"] = sata0
	}
	if net0 := request.GetString("net0", ""); net0 != "" {
		config["net0"] = net0
	}

	var result interface{}
	var err error
	name := request.GetString("name", "")
	hasAdvanced := request.GetString("ide2", "") != "" || request.GetString("sata0", "") != "" || request.GetString("net0", "") != ""
	if name != "" && !hasAdvanced {
		mem := request.GetInt("memory", 512)
		cores := request.GetInt("cores", 1)
		sockets := request.GetInt("sockets", 1)
		result, err = s.proxmoxClient.CreateVMFull(ctx, nodeName, vmID, name, mem, cores, sockets)
	} else {
		result, err = s.proxmoxClient.CreateVM(ctx, nodeName, config)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "create",
		"vmid":   vmID,
		"node":   nodeName,
		"config": config,
		"result": result,
	})
}

// cloneVM handles the clone_vm tool
func (s *Server) cloneVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: clone_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	sourceVMID := request.GetInt("source_vmid", 0)
	if sourceVMID <= 0 {
		return mcp.NewToolResultError("source_vmid parameter is required and must be a positive integer"), nil
	}

	newVMID := request.GetInt("new_vmid", 0)
	if newVMID <= 0 {
		return mcp.NewToolResultError("new_vmid parameter is required and must be a positive integer"), nil
	}

	newName := request.GetString("new_name", "")
	if newName == "" {
		return mcp.NewToolResultError("new_name parameter is required"), nil
	}

	full := request.GetBool("full", true)

	result, err := s.proxmoxClient.CloneVM(ctx, nodeName, sourceVMID, newVMID, newName, full)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to clone VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":      "clone",
		"source_vmid": sourceVMID,
		"new_vmid":    newVMID,
		"new_name":    newName,
		"node":        nodeName,
		"full_clone":  full,
		"result":      result,
	})
}

// cloneGuest handles the clone_guest tool (VM or LXC)
func (s *Server) cloneGuest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: clone_guest")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	if guestType != "vm" && guestType != "lxc" {
		return mcp.NewToolResultError("guest_type must be vm or lxc"), nil
	}
	sourceID := request.GetInt("source_vmid", 0)
	if sourceID <= 0 {
		return mcp.NewToolResultError("source_vmid parameter is required and must be a positive integer"), nil
	}
	newID := request.GetInt("new_vmid", 0)
	if newID <= 0 {
		return mcp.NewToolResultError("new_vmid parameter is required and must be a positive integer"), nil
	}
	newName := request.GetString("new_name", "")
	if newName == "" {
		return mcp.NewToolResultError("new_name parameter is required (VM name or container hostname)"), nil
	}
	full := request.GetBool("full", true)

	var result interface{}
	var err error
	if guestType == "vm" {
		result, err = s.proxmoxClient.CloneVM(ctx, nodeName, sourceID, newID, newName, full)
	} else {
		result, err = s.proxmoxClient.CloneContainer(ctx, nodeName, sourceID, newID, newName, full)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to clone %s: %v", guestType, err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":      "clone",
		"guest_type":  guestType,
		"source_vmid": sourceID,
		"new_vmid":    newID,
		"new_name":   newName,
		"node":        nodeName,
		"full_clone":  full,
		"result":      result,
	})
}

// updateGuestConfig handles the update_guest_config tool (flat params for VM or container)
func (s *Server) updateGuestConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: update_guest_config")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	if guestType != "vm" && guestType != "lxc" {
		return mcp.NewToolResultError("guest_type must be vm or lxc"), nil
	}
	vmid := request.GetInt("vmid", 0)
	if vmid <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	config := make(map[string]interface{})
	if v := request.GetInt("memory", 0); v > 0 {
		config["memory"] = v
	}
	if v := request.GetInt("cores", 0); v > 0 {
		config["cores"] = v
	}
	if v := request.GetInt("sockets", 0); v > 0 {
		config["sockets"] = v
	}
	if v := request.GetString("name", ""); v != "" {
		config["name"] = v
	}
	if v := request.GetString("hostname", ""); v != "" {
		config["hostname"] = v
	}
	if v := request.GetInt("template", -1); v >= 0 {
		config["template"] = v
	}
	if v := request.GetString("net0", ""); v != "" {
		config["net0"] = v
	}
	if len(config) == 0 {
		return mcp.NewToolResultError("at least one optional config parameter is required (memory, cores, name, hostname, template, net0, sockets)"), nil
	}

	var result interface{}
	var err error
	if guestType == "vm" {
		result, err = s.proxmoxClient.UpdateVM(ctx, nodeName, vmid, config)
	} else {
		result, err = s.proxmoxClient.UpdateContainer(ctx, nodeName, vmid, config)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update %s config: %v", guestType, err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":     "update_config",
		"guest_type": guestType,
		"vmid":       vmid,
		"node":       nodeName,
		"config":    config,
		"result":    result,
	})
}

// updateVMConfig handles the update_vm_config tool (legacy; prefer update_guest_config)
func (s *Server) updateVMConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: update_vm_config")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	args := request.GetArguments()
	configValue := args["config"]
	if configValue == nil {
		return mcp.NewToolResultError("config parameter is required"), nil
	}

	var config map[string]interface{}
	configBytes, err := json.Marshal(configValue)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid config parameter: %v", err)), nil
	}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse config: %v", err)), nil
	}

	result, err := s.proxmoxClient.UpdateVM(ctx, nodeName, vmID, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update VM config: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "update_config",
		"vmid":   vmID,
		"node":   nodeName,
		"config": config,
		"result": result,
	})
}

// getVMConsole handles the get_vm_console tool
func (s *Server) getVMConsole(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_vm_console")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.GetVMConsole(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get VM console: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"vmid":    vmID,
		"node":    nodeName,
		"console": result,
	})
}

// createSnapshot handles the create_snapshot tool (VM or container)
func (s *Server) createSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_snapshot")
	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	if guestType != "vm" && guestType != "lxc" {
		return mcp.NewToolResultError("guest_type must be vm or lxc"), nil
	}
	vmid := request.GetInt("vmid", 0)
	if vmid <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}
	snapName := request.GetString("snap_name", "")
	if snapName == "" {
		return mcp.NewToolResultError("snap_name parameter is required"), nil
	}
	description := request.GetString("description", "")
	var result interface{}
	var err error
	if guestType == "vm" {
		result, err = s.proxmoxClient.CreateVMSnapshot(ctx, nodeName, vmid, snapName, description)
	} else {
		result, err = s.proxmoxClient.CreateContainerSnapshot(ctx, nodeName, vmid, snapName, description)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create snapshot: %v", err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "create_snapshot", "guest_type": guestType, "vmid": vmid, "node": nodeName,
		"snapshot": snapName, "description": description, "result": result,
	})
}

// listSnapshots handles the list_snapshots tool (VM or container)
func (s *Server) listSnapshots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_snapshots")
	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	if guestType != "vm" && guestType != "lxc" {
		return mcp.NewToolResultError("guest_type must be vm or lxc"), nil
	}
	vmid := request.GetInt("vmid", 0)
	if vmid <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}
	var snapshots interface{}
	var err error
	if guestType == "vm" {
		snapshots, err = s.proxmoxClient.ListVMSnapshots(ctx, nodeName, vmid)
	} else {
		snapshots, err = s.proxmoxClient.ListContainerSnapshots(ctx, nodeName, vmid)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list snapshots: %v", err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{
		"guest_type": guestType, "vmid": vmid, "node": nodeName, "snapshots": snapshots,
	})
}

// deleteSnapshot handles the delete_snapshot tool (VM or container; requires confirm)
func (s *Server) deleteSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_snapshot")
	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	if guestType != "vm" && guestType != "lxc" {
		return mcp.NewToolResultError("guest_type must be vm or lxc"), nil
	}
	vmid := request.GetInt("vmid", 0)
	if vmid <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}
	snapName := request.GetString("snap_name", "")
	if snapName == "" {
		return mcp.NewToolResultError("snap_name parameter is required"), nil
	}
	expectedConfirm := fmt.Sprintf("%s/%d/%s", nodeName, vmid, snapName)
	if request.GetString("confirm", "") != expectedConfirm {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", expectedConfirm)), nil
	}
	force := request.GetBool("force", false)
	var result interface{}
	var err error
	if guestType == "vm" {
		result, err = s.proxmoxClient.DeleteVMSnapshot(ctx, nodeName, vmid, snapName, force)
	} else {
		result, err = s.proxmoxClient.DeleteContainerSnapshot(ctx, nodeName, vmid, snapName, force)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete snapshot: %v", err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "delete_snapshot", "guest_type": guestType, "vmid": vmid, "node": nodeName,
		"snapshot": snapName, "result": result,
	})
}

// restoreSnapshot handles the restore_snapshot tool (VM or container)
func (s *Server) restoreSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: restore_snapshot")
	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}
	guestType := strings.ToLower(request.GetString("guest_type", "vm"))
	if guestType != "vm" && guestType != "lxc" {
		return mcp.NewToolResultError("guest_type must be vm or lxc"), nil
	}
	vmid := request.GetInt("vmid", 0)
	if vmid <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}
	snapName := request.GetString("snap_name", "")
	if snapName == "" {
		return mcp.NewToolResultError("snap_name parameter is required"), nil
	}
	var result interface{}
	var err error
	if guestType == "vm" {
		result, err = s.proxmoxClient.RestoreVMSnapshot(ctx, nodeName, vmid, snapName)
	} else {
		result, err = s.proxmoxClient.RestoreContainerSnapshot(ctx, nodeName, vmid, snapName)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to restore snapshot: %v", err)), nil
	}
	return mcp.NewToolResultJSON(map[string]interface{}{
		"action": "restore_snapshot", "guest_type": guestType, "vmid": vmid, "node": nodeName,
		"snapshot": snapName, "result": result,
	})
}

// createVMSnapshot handles the create_vm_snapshot tool
func (s *Server) createVMSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_vm_snapshot")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	snapName := request.GetString("snap_name", "")
	if snapName == "" {
		return mcp.NewToolResultError("snap_name parameter is required"), nil
	}

	description := request.GetString("description", "")

	result, err := s.proxmoxClient.CreateVMSnapshot(ctx, nodeName, vmID, snapName, description)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create VM snapshot: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":      "create_snapshot",
		"vmid":        vmID,
		"node":        nodeName,
		"snapshot":    snapName,
		"description": description,
		"result":      result,
	})
}

// listVMSnapshots handles the list_vm_snapshots tool
func (s *Server) listVMSnapshots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_vm_snapshots")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.ListVMSnapshots(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list VM snapshots: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"vmid":      vmID,
		"node":      nodeName,
		"snapshots": result,
	})
}

// deleteVMSnapshot handles the delete_vm_snapshot tool
func (s *Server) deleteVMSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_vm_snapshot")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	snapName := request.GetString("snap_name", "")
	if snapName == "" {
		return mcp.NewToolResultError("snap_name parameter is required"), nil
	}

	force := request.GetBool("force", false)

	result, err := s.proxmoxClient.DeleteVMSnapshot(ctx, nodeName, vmID, snapName, force)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete VM snapshot: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":   "delete_snapshot",
		"vmid":     vmID,
		"node":     nodeName,
		"snapshot": snapName,
		"force":    force,
		"result":   result,
	})
}

// restoreVMSnapshot handles the restore_vm_snapshot tool
func (s *Server) restoreVMSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: restore_vm_snapshot")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	snapName := request.GetString("snap_name", "")
	if snapName == "" {
		return mcp.NewToolResultError("snap_name parameter is required"), nil
	}

	result, err := s.proxmoxClient.RestoreVMSnapshot(ctx, nodeName, vmID, snapName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to restore VM snapshot: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":   "restore_snapshot",
		"vmid":     vmID,
		"node":     nodeName,
		"snapshot": snapName,
		"result":   result,
	})
}

// getVMFirewallRules handles the get_vm_firewall_rules tool
func (s *Server) getVMFirewallRules(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_vm_firewall_rules")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.GetVMFirewallRules(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get VM firewall rules: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"vmid":  vmID,
		"node":  nodeName,
		"rules": result,
	})
}

// migrateVM handles the migrate_vm tool
func (s *Server) migrateVM(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: migrate_vm")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	targetNode := request.GetString("target_node", "")
	if targetNode == "" {
		return mcp.NewToolResultError("target_node parameter is required"), nil
	}

	online := request.GetBool("online", false)

	result, err := s.proxmoxClient.MigrateVM(ctx, nodeName, vmID, targetNode, online)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to migrate VM: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":      "migrate",
		"vmid":        vmID,
		"source_node": nodeName,
		"target_node": targetNode,
		"online":      online,
		"result":      result,
	})
}

// startContainer handles the start_container tool
func (s *Server) startContainer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: start_container")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.StartContainer(ctx, nodeName, containerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to start container: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "start",
		"container_id": containerID,
		"node":         nodeName,
		"result":       result,
	})
}

// stopContainer handles the stop_container tool
func (s *Server) stopContainer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: stop_container")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.StopContainer(ctx, nodeName, containerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to stop container: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "stop",
		"container_id": containerID,
		"node":         nodeName,
		"result":       result,
	})
}

// shutdownContainer handles the shutdown_container tool
func (s *Server) shutdownContainer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: shutdown_container")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.ShutdownContainer(ctx, nodeName, containerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to shutdown container: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "shutdown",
		"container_id": containerID,
		"node":         nodeName,
		"result":       result,
	})
}

// rebootContainer handles the reboot_container tool
func (s *Server) rebootContainer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: reboot_container")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.RebootContainer(ctx, nodeName, containerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to reboot container: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "reboot",
		"container_id": containerID,
		"node":         nodeName,
		"result":       result,
	})
}

// getContainerConfig handles the get_container_config tool
func (s *Server) getContainerConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_container_config")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	config, err := s.proxmoxClient.GetContainerConfig(ctx, nodeName, containerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get container config: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"container_id": containerID,
		"node":         nodeName,
		"config":       config,
	})
}

// deleteContainer handles the delete_container tool
func (s *Server) deleteContainer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_container")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	expectedConfirm := fmt.Sprintf("%s/%d", nodeName, containerID)
	if request.GetString("confirm", "") != expectedConfirm {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", expectedConfirm)), nil
	}

	force := request.GetBool("force", false)

	result, err := s.proxmoxClient.DeleteContainer(ctx, nodeName, containerID, force)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete container: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "delete",
		"container_id": containerID,
		"node":         nodeName,
		"force":        force,
		"result":       result,
	})
}

// createContainer handles the create_container tool (single tool with all optional flat params)
func (s *Server) createContainer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_container")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		containerID = request.GetInt("vmid", 0)
	}
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id or vmid is required and must be a positive integer"), nil
	}

	config := map[string]interface{}{"vmid": containerID}
	if hostname := request.GetString("hostname", ""); hostname != "" {
		config["hostname"] = hostname
	}
	if storage := request.GetString("storage", ""); storage != "" {
		config["storage"] = storage
	}
	if memory := request.GetInt("memory", 0); memory > 0 {
		config["memory"] = memory
	}
	if cores := request.GetInt("cores", 0); cores > 0 {
		config["cores"] = cores
	}
	if ostype := request.GetString("ostype", ""); ostype != "" {
		config["ostype"] = ostype
	}
	if net0 := request.GetString("net0", ""); net0 != "" {
		config["net0"] = net0
	}
	if rootfs := request.GetString("rootfs", ""); rootfs != "" {
		config["rootfs"] = rootfs
	}

	hostname := request.GetString("hostname", "")
	storage := request.GetString("storage", "")
	hasAdvanced := request.GetString("net0", "") != "" || request.GetString("rootfs", "") != ""
	var result interface{}
	var err error
	if hostname != "" && storage != "" && !hasAdvanced {
		memory := request.GetInt("memory", 512)
		cores := request.GetInt("cores", 1)
		ostype := request.GetString("ostype", "debian")
		result, err = s.proxmoxClient.CreateContainerFull(ctx, nodeName, containerID, hostname, storage, memory, cores, ostype)
	} else {
		result, err = s.proxmoxClient.CreateContainer(ctx, nodeName, config)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create container: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "create",
		"container_id": containerID,
		"node":         nodeName,
		"config":       config,
		"result":       result,
	})
}

// cloneContainer handles the clone_container tool
func (s *Server) cloneContainer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: clone_container")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	sourceContainerID := request.GetInt("source_container_id", 0)
	if sourceContainerID <= 0 {
		return mcp.NewToolResultError("source_container_id parameter is required and must be a positive integer"), nil
	}

	newContainerID := request.GetInt("new_container_id", 0)
	if newContainerID <= 0 {
		return mcp.NewToolResultError("new_container_id parameter is required and must be a positive integer"), nil
	}

	newHostname := request.GetString("new_hostname", "")
	if newHostname == "" {
		return mcp.NewToolResultError("new_hostname parameter is required"), nil
	}

	full := request.GetBool("full", true)

	result, err := s.proxmoxClient.CloneContainer(ctx, nodeName, sourceContainerID, newContainerID, newHostname, full)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to clone container: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":              "clone",
		"source_container_id": sourceContainerID,
		"new_container_id":    newContainerID,
		"new_hostname":        newHostname,
		"node":                nodeName,
		"full_clone":          full,
		"result":              result,
	})
}

// updateContainerConfig handles the update_container_config tool
func (s *Server) updateContainerConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: update_container_config")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	// Get all arguments to extract config
	args := request.GetArguments()
	configValue := args["config"]
	if configValue == nil {
		return mcp.NewToolResultError("config parameter is required"), nil
	}

	// Convert config to map[string]interface{}
	var config map[string]interface{}
	configBytes, err := json.Marshal(configValue)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid config parameter: %v", err)), nil
	}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse config: %v", err)), nil
	}

	result, err := s.proxmoxClient.UpdateContainer(ctx, nodeName, containerID, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update container config: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "update_config",
		"container_id": containerID,
		"node":         nodeName,
		"config":       config,
		"result":       result,
	})
}

// createContainerSnapshot handles the create_container_snapshot tool
func (s *Server) createContainerSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_container_snapshot")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	snapName := request.GetString("snap_name", "")
	if snapName == "" {
		return mcp.NewToolResultError("snap_name parameter is required"), nil
	}

	description := request.GetString("description", "")

	result, err := s.proxmoxClient.CreateContainerSnapshot(ctx, nodeName, containerID, snapName, description)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create container snapshot: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "create_snapshot",
		"container_id": containerID,
		"node":         nodeName,
		"snapshot":     snapName,
		"description":  description,
		"result":       result,
	})
}

// listContainerSnapshots handles the list_container_snapshots tool
func (s *Server) listContainerSnapshots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_container_snapshots")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	result, err := s.proxmoxClient.ListContainerSnapshots(ctx, nodeName, containerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list container snapshots: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"container_id": containerID,
		"node":         nodeName,
		"snapshots":    result,
	})
}

// deleteContainerSnapshot handles the delete_container_snapshot tool
func (s *Server) deleteContainerSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_container_snapshot")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	snapName := request.GetString("snap_name", "")
	if snapName == "" {
		return mcp.NewToolResultError("snap_name parameter is required"), nil
	}

	force := request.GetBool("force", false)

	result, err := s.proxmoxClient.DeleteContainerSnapshot(ctx, nodeName, containerID, snapName, force)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete container snapshot: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "delete_snapshot",
		"container_id": containerID,
		"node":         nodeName,
		"snapshot":     snapName,
		"force":        force,
		"result":       result,
	})
}

// restoreContainerSnapshot handles the restore_container_snapshot tool
func (s *Server) restoreContainerSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: restore_container_snapshot")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	snapName := request.GetString("snap_name", "")
	if snapName == "" {
		return mcp.NewToolResultError("snap_name parameter is required"), nil
	}

	result, err := s.proxmoxClient.RestoreContainerSnapshot(ctx, nodeName, containerID, snapName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to restore container snapshot: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "restore_snapshot",
		"container_id": containerID,
		"node":         nodeName,
		"snapshot":     snapName,
		"result":       result,
	})
}

// ============ USER MANAGEMENT ============

// listUsers handles the list_users tool
func (s *Server) listUsers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_users")

	users, err := s.proxmoxClient.ListUsers(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list users: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

// getUser handles the get_user tool
func (s *Server) getUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_user")

	userID := request.GetString("userid", "")
	if userID == "" {
		return mcp.NewToolResultError("userid parameter is required"), nil
	}

	user, err := s.proxmoxClient.GetUser(ctx, userID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get user: %v", err)), nil
	}

	// Ensure userid is set (Proxmox API doesn't always return it in the response)
	if user.UserID == "" {
		user.UserID = userID
	}

	return mcp.NewToolResultJSON(user)
}

// createUser handles the create_user tool
func (s *Server) createUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_user")

	userID := request.GetString("userid", "")
	if userID == "" {
		return mcp.NewToolResultError("userid parameter is required"), nil
	}

	password := request.GetString("password", "")
	email := request.GetString("email", "")
	comment := request.GetString("comment", "")

	result, err := s.proxmoxClient.CreateUser(ctx, userID, password, email, comment)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create user: %v", err)), nil
	}

	message := "User created successfully"
	if password != "" {
		// Check if password was actually set by checking if there was an error
		// The client will attempt to set password, but it may fail with API tokens
		message += " (Password setting attempted - may fail with API token for PAM realm, works with session ticket)"
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "create",
		"userid":  userID,
		"message": message,
		"result":  result,
	})
}

// updateUser handles the update_user tool
func (s *Server) updateUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: update_user")

	userID := request.GetString("userid", "")
	if userID == "" {
		return mcp.NewToolResultError("userid parameter is required"), nil
	}

	email := request.GetString("email", "")
	comment := request.GetString("comment", "")
	firstName := request.GetString("firstname", "")
	lastName := request.GetString("lastname", "")
	enable := request.GetBool("enable", true)
	expire := int64(request.GetInt("expire", 0))

	result, err := s.proxmoxClient.UpdateUser(ctx, userID, email, comment, firstName, lastName, enable, expire)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update user: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "update",
		"userid":  userID,
		"message": "User updated successfully",
		"result":  result,
	})
}

// deleteUser handles the delete_user tool
func (s *Server) deleteUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_user")

	userID := request.GetString("userid", "")
	if userID == "" {
		return mcp.NewToolResultError("userid parameter is required"), nil
	}
	if request.GetString("confirm", "") != userID {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", userID)), nil
	}

	result, err := s.proxmoxClient.DeleteUser(ctx, userID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete user: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "delete",
		"userid":  userID,
		"message": "User deleted successfully",
		"result":  result,
	})
}

// changePassword handles the change_password tool
func (s *Server) changePassword(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: change_password")

	userID := request.GetString("userid", "")
	if userID == "" {
		return mcp.NewToolResultError("userid parameter is required"), nil
	}

	password := request.GetString("password", "")
	if password == "" {
		return mcp.NewToolResultError("password parameter is required"), nil
	}

	result, err := s.proxmoxClient.ChangePassword(ctx, userID, password)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to change password: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "change_password",
		"userid":  userID,
		"message": "Password changed successfully",
		"result":  result,
	})
}

// listGroups handles the list_groups tool
func (s *Server) listGroups(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_groups")

	groups, err := s.proxmoxClient.ListGroups(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list groups: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"groups": groups,
		"count":  len(groups),
	})
}

// createGroup handles the create_group tool
func (s *Server) createGroup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_group")

	groupID := request.GetString("groupid", "")
	if groupID == "" {
		return mcp.NewToolResultError("groupid parameter is required"), nil
	}

	comment := request.GetString("comment", "")

	result, err := s.proxmoxClient.CreateGroup(ctx, groupID, comment)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create group: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "create",
		"groupid": groupID,
		"message": "Group created successfully",
		"result":  result,
	})
}

// deleteGroup handles the delete_group tool
func (s *Server) deleteGroup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_group")

	groupID := request.GetString("groupid", "")
	if groupID == "" {
		return mcp.NewToolResultError("groupid parameter is required"), nil
	}
	if request.GetString("confirm", "") != groupID {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", groupID)), nil
	}

	result, err := s.proxmoxClient.DeleteGroup(ctx, groupID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete group: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "delete",
		"groupid": groupID,
		"message": "Group deleted successfully",
		"result":  result,
	})
}

// listRoles handles the list_roles tool
func (s *Server) listRoles(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_roles")

	roles, err := s.proxmoxClient.ListRoles(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list roles: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"roles": roles,
		"count": len(roles),
	})
}

// createRole handles the create_role tool
func (s *Server) createRole(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_role")

	roleID := request.GetString("roleid", "")
	if roleID == "" {
		return mcp.NewToolResultError("roleid parameter is required"), nil
	}

	// Parse privileges - for simplicity, accept a space-separated string or array
	privs := []string{}

	// Try to get as string first (space-separated)
	if privStr := request.GetString("privs", ""); privStr != "" {
		// If it's a string, it might be space-separated
		privsList := splitPrivileges(privStr)
		privs = privsList
	}

	result, err := s.proxmoxClient.CreateRole(ctx, roleID, privs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create role: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "create",
		"roleid":  roleID,
		"message": "Role created successfully",
		"result":  result,
	})
}

// Helper function to split privileges string
func splitPrivileges(privStr string) []string {
	if privStr == "" {
		return []string{}
	}
	// For now, assume space-separated or comma-separated
	parts := make([]string, 0)
	current := ""
	for _, ch := range privStr {
		if ch == ' ' || ch == ',' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// deleteRole handles the delete_role tool
func (s *Server) deleteRole(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_role")

	roleID := request.GetString("roleid", "")
	if roleID == "" {
		return mcp.NewToolResultError("roleid parameter is required"), nil
	}
	if request.GetString("confirm", "") != roleID {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", roleID)), nil
	}

	result, err := s.proxmoxClient.DeleteRole(ctx, roleID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete role: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "delete",
		"roleid":  roleID,
		"message": "Role deleted successfully",
		"result":  result,
	})
}

// listACLs handles the list_acl tool
func (s *Server) listACLs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_acl")

	acls, err := s.proxmoxClient.ListACLs(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list ACLs: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"acls":  acls,
		"count": len(acls),
	})
}

// setACL handles the set_acl tool
func (s *Server) setACL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: set_acl")

	path := request.GetString("path", "")
	if path == "" {
		return mcp.NewToolResultError("path parameter is required"), nil
	}

	role := request.GetString("role", "")
	if role == "" {
		return mcp.NewToolResultError("role parameter is required"), nil
	}

	userID := request.GetString("userid", "")
	groupID := request.GetString("groupid", "")
	tokenID := request.GetString("tokenid", "")
	propagate := request.GetBool("propagate", true)

	if userID == "" && groupID == "" && tokenID == "" {
		return mcp.NewToolResultError("At least one of userid, groupid, or tokenid is required"), nil
	}

	result, err := s.proxmoxClient.SetACL(ctx, path, role, userID, groupID, tokenID, propagate)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to set ACL: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":    "set_acl",
		"path":      path,
		"role":      role,
		"propagate": propagate,
		"message":   "ACL set successfully",
		"result":    result,
	})
}

// createAPIToken handles the create_api_token tool
func (s *Server) createAPIToken(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_api_token")

	userID := request.GetString("userid", "")
	if userID == "" {
		return mcp.NewToolResultError("userid parameter is required"), nil
	}

	tokenID := request.GetString("tokenid", "")
	if tokenID == "" {
		return mcp.NewToolResultError("tokenid parameter is required"), nil
	}

	expire := int64(request.GetInt("expire", 0))
	privSep := request.GetBool("privsep", false)

	result, err := s.proxmoxClient.CreateAPIToken(ctx, userID, tokenID, expire, privSep)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create API token: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "create",
		"userid":  userID,
		"tokenid": tokenID,
		"message": "API token created successfully",
		"result":  result,
	})
}

// deleteAPIToken handles the delete_api_token tool
func (s *Server) deleteAPIToken(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_api_token")

	userID := request.GetString("userid", "")
	if userID == "" {
		return mcp.NewToolResultError("userid parameter is required"), nil
	}

	tokenID := request.GetString("tokenid", "")
	if tokenID == "" {
		return mcp.NewToolResultError("tokenid parameter is required"), nil
	}
	expectedConfirm := fmt.Sprintf("%s/%s", userID, tokenID)
	if request.GetString("confirm", "") != expectedConfirm {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", expectedConfirm)), nil
	}

	result, err := s.proxmoxClient.DeleteAPIToken(ctx, userID, tokenID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete API token: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "delete",
		"userid":  userID,
		"tokenid": tokenID,
		"message": "API token deleted successfully",
		"result":  result,
	})
}

// ============ BACKUP & RESTORE ============

// listBackups handles the list_backups tool
func (s *Server) listBackups(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_backups")

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	backups, err := s.proxmoxClient.ListBackups(ctx, storage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list backups: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"backups": backups,
		"storage": storage,
		"count":   len(backups),
	})
}

// createVMBackup handles the create_vm_backup tool
func (s *Server) createVMBackup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_vm_backup")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID <= 0 {
		return mcp.NewToolResultError("vmid parameter is required and must be a positive integer"), nil
	}

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	backupID := request.GetString("backup_id", "")
	notes := request.GetString("notes", "")

	result, err := s.proxmoxClient.CreateVMBackup(ctx, nodeName, vmID, storage, backupID, notes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create VM backup: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":  "backup",
		"vmid":    vmID,
		"node":    nodeName,
		"storage": storage,
		"message": "VM backup started",
		"result":  result,
	})
}

// createContainerBackup handles the create_container_backup tool
func (s *Server) createContainerBackup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_container_backup")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID <= 0 {
		return mcp.NewToolResultError("container_id parameter is required and must be a positive integer"), nil
	}

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	backupID := request.GetString("backup_id", "")
	notes := request.GetString("notes", "")

	result, err := s.proxmoxClient.CreateContainerBackup(ctx, nodeName, containerID, storage, backupID, notes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create container backup: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":       "backup",
		"container_id": containerID,
		"node":         nodeName,
		"storage":      storage,
		"message":      "Container backup started",
		"result":       result,
	})
}

// deleteBackup handles the delete_backup tool
func (s *Server) deleteBackup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_backup")

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	backupID := request.GetString("backup_id", "")
	if backupID == "" {
		return mcp.NewToolResultError("backup_id parameter is required"), nil
	}
	expectedConfirm := fmt.Sprintf("%s/%s", storage, backupID)
	if request.GetString("confirm", "") != expectedConfirm {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", expectedConfirm)), nil
	}

	result, err := s.proxmoxClient.DeleteBackup(ctx, storage, backupID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete backup: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":    "delete",
		"storage":   storage,
		"backup_id": backupID,
		"message":   "Backup deleted successfully",
		"result":    result,
	})
}

// restoreVMBackup handles the restore_vm_backup tool
func (s *Server) restoreVMBackup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: restore_vm_backup")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	backupID := request.GetString("backup_id", "")
	if backupID == "" {
		return mcp.NewToolResultError("backup_id parameter is required"), nil
	}

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	result, err := s.proxmoxClient.RestoreVMBackup(ctx, nodeName, backupID, storage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to restore VM backup: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":    "restore",
		"node":      nodeName,
		"backup_id": backupID,
		"storage":   storage,
		"message":   "VM restore started",
		"result":    result,
	})
}

// restoreContainerBackup handles the restore_container_backup tool
func (s *Server) restoreContainerBackup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: restore_container_backup")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	backupID := request.GetString("backup_id", "")
	if backupID == "" {
		return mcp.NewToolResultError("backup_id parameter is required"), nil
	}

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	result, err := s.proxmoxClient.RestoreContainerBackup(ctx, nodeName, backupID, storage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to restore container backup: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"action":    "restore",
		"node":      nodeName,
		"backup_id": backupID,
		"storage":   storage,
		"message":   "Container restore started",
		"result":    result,
	})
}

// ============ RESOURCE POOLS ============

func (s *Server) listPools(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: list_pools")

	pools, err := s.proxmoxClient.ListPools(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list pools: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Pools retrieved successfully",
		"pools":   pools,
	})
}

func (s *Server) getPool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_pool")

	poolID := request.GetString("poolid", "")
	if poolID == "" {
		return mcp.NewToolResultError("poolid parameter is required"), nil
	}

	pool, err := s.proxmoxClient.GetPool(ctx, poolID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get pool: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Pool retrieved successfully",
		"pool":    pool,
	})
}

// ============ NODE MANAGEMENT - TASKS ============

func (s *Server) getNodeTasks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_tasks")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	tasks, err := s.proxmoxClient.GetNodeTasks(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node tasks: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node tasks retrieved successfully",
		"node":    nodeName,
		"tasks":   tasks,
	})
}

func (s *Server) getClusterTasks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_cluster_tasks")

	tasks, err := s.proxmoxClient.GetClusterTasks(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get cluster tasks: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Cluster tasks retrieved successfully",
		"tasks":   tasks,
	})
}

// ============ STATISTICS ============

func (s *Server) getNodeStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_stats")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	stats, err := s.proxmoxClient.GetNodeStats(ctx, nodeName, "day")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node statistics: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node statistics retrieved successfully",
		"node":    nodeName,
		"stats":   stats,
	})
}

func (s *Server) getVMStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_vm_stats")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vmID := request.GetInt("vmid", 0)
	if vmID == 0 {
		return mcp.NewToolResultError("vmid parameter is required"), nil
	}

	stats, err := s.proxmoxClient.GetVMStats(ctx, nodeName, vmID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get VM statistics: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "VM statistics retrieved successfully",
		"node":    nodeName,
		"vmid":    vmID,
		"stats":   stats,
	})
}

func (s *Server) getContainerStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_container_stats")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	containerID := request.GetInt("container_id", 0)
	if containerID == 0 {
		return mcp.NewToolResultError("container_id parameter is required"), nil
	}

	stats, err := s.proxmoxClient.GetContainerStats(ctx, nodeName, containerID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get container statistics: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message":      "Container statistics retrieved successfully",
		"node":         nodeName,
		"container_id": containerID,
		"stats":        stats,
	})
}

// ============ PHASE 4: STORAGE MANAGEMENT ============

func (s *Server) getStorageInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_storage_info")

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	info, err := s.proxmoxClient.GetStorageInfo(ctx, storage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get storage info: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Storage information retrieved successfully",
		"storage": storage,
		"info":    info,
	})
}

func (s *Server) createStorage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_storage")

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	storageType := request.GetString("storage_type", "")
	if storageType == "" {
		return mcp.NewToolResultError("storage_type parameter is required"), nil
	}

	content := request.GetString("content", "")
	if content == "" {
		return mcp.NewToolResultError("content parameter is required"), nil
	}

	// Get config parameter (optional)
	config := make(map[string]interface{})
	args := request.GetArguments()
	if configValue, ok := args["config"]; ok && configValue != nil {
		configBytes, err := json.Marshal(configValue)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid config parameter: %v", err)), nil
		}
		err = json.Unmarshal(configBytes, &config)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse config: %v", err)), nil
		}
	}

	result, err := s.proxmoxClient.CreateStorage(ctx, storage, storageType, content, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create storage: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Storage created successfully",
		"storage": storage,
		"result":  result,
	})
}

func (s *Server) deleteStorage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_storage")

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}
	if request.GetString("confirm", "") != storage {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", storage)), nil
	}

	result, err := s.proxmoxClient.DeleteStorage(ctx, storage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete storage: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Storage deleted successfully",
		"storage": storage,
		"result":  result,
	})
}

func (s *Server) updateStorage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: update_storage")

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	args := request.GetArguments()
	configValue := args["config"]
	if configValue == nil {
		return mcp.NewToolResultError("config parameter is required"), nil
	}

	// Convert config to map[string]interface{}
	var config map[string]interface{}
	configBytes, err := json.Marshal(configValue)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid config parameter: %v", err)), nil
	}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse config: %v", err)), nil
	}

	result, err := s.proxmoxClient.UpdateStorage(ctx, storage, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update storage: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Storage updated successfully",
		"storage": storage,
		"result":  result,
	})
}

func (s *Server) getStorageContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_storage_content")

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	content, err := s.proxmoxClient.GetStorageContent(ctx, storage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get storage content: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Storage content retrieved successfully",
		"storage": storage,
		"content": content,
	})
}

// ============ PHASE 4: TASK MANAGEMENT ============

func (s *Server) getTaskStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_task_status")

	taskID := request.GetString("task_id", "")
	if taskID == "" {
		return mcp.NewToolResultError("task_id parameter is required"), nil
	}

	status, err := s.proxmoxClient.GetTaskStatus(ctx, taskID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get task status: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Task status retrieved successfully",
		"task_id": taskID,
		"status":  status,
	})
}

func (s *Server) getTaskLog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_task_log")

	taskID := request.GetString("task_id", "")
	if taskID == "" {
		return mcp.NewToolResultError("task_id parameter is required"), nil
	}

	start := request.GetInt("start", 0)
	limit := request.GetInt("limit", 50)

	log, err := s.proxmoxClient.GetTaskLog(ctx, taskID, start, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get task log: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Task log retrieved successfully",
		"task_id": taskID,
		"log":     log,
	})
}

func (s *Server) cancelTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: cancel_task")

	taskID := request.GetString("task_id", "")
	if taskID == "" {
		return mcp.NewToolResultError("task_id parameter is required"), nil
	}

	result, err := s.proxmoxClient.CancelTask(ctx, taskID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to cancel task: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Task cancellation requested",
		"task_id": taskID,
		"result":  result,
	})
}

// ============ PHASE 4: NODE MANAGEMENT ============

func (s *Server) getNodeConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_config")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	config, err := s.proxmoxClient.GetNodeConfig(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node config: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node configuration retrieved successfully",
		"node":    nodeName,
		"config":  config,
	})
}

func (s *Server) updateNodeConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: update_node_config")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	args := request.GetArguments()
	configValue := args["config"]
	if configValue == nil {
		return mcp.NewToolResultError("config parameter is required"), nil
	}

	// Convert config to map[string]interface{}
	var config map[string]interface{}
	configBytes, err := json.Marshal(configValue)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid config parameter: %v", err)), nil
	}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse config: %v", err)), nil
	}

	result, err := s.proxmoxClient.UpdateNodeConfig(ctx, nodeName, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update node config: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node configuration updated successfully",
		"node":    nodeName,
		"result":  result,
	})
}

func (s *Server) rebootNode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: reboot_node")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	result, err := s.proxmoxClient.RebootNode(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to reboot node: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node reboot initiated",
		"node":    nodeName,
		"result":  result,
	})
}

func (s *Server) shutdownNode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: shutdown_node")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	result, err := s.proxmoxClient.ShutdownNode(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to shutdown node: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node shutdown initiated",
		"node":    nodeName,
		"result":  result,
	})
}

func (s *Server) getNodeDisks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_disks")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	disks, err := s.proxmoxClient.GetNodeDisks(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node disks: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node disks retrieved successfully",
		"node":    nodeName,
		"disks":   disks,
	})
}

func (s *Server) getNodeCert(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_cert")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	cert, err := s.proxmoxClient.GetNodeCert(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node certificate: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node certificate information retrieved successfully",
		"node":    nodeName,
		"cert":    cert,
	})
}

// ============ POOL MANAGEMENT ============

func (s *Server) createPool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_pool")

	poolID := request.GetString("poolid", "")
	if poolID == "" {
		return mcp.NewToolResultError("poolid parameter is required"), nil
	}

	comment := request.GetString("comment", "")

	// Get members array
	args := request.GetArguments()
	members := []string{}
	if membersValue, ok := args["members"]; ok && membersValue != nil {
		membersBytes, err := json.Marshal(membersValue)
		if err == nil {
			json.Unmarshal(membersBytes, &members)
		}
	}

	result, err := s.proxmoxClient.CreatePool(ctx, poolID, comment, members)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create pool: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Pool created successfully",
		"poolid":  poolID,
		"result":  result,
	})
}

func (s *Server) updatePool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: update_pool")

	poolID := request.GetString("poolid", "")
	if poolID == "" {
		return mcp.NewToolResultError("poolid parameter is required"), nil
	}

	comment := request.GetString("comment", "")
	delete := request.GetBool("delete", false)

	// Get members array
	args := request.GetArguments()
	members := []string{}
	if membersValue, ok := args["members"]; ok && membersValue != nil {
		membersBytes, err := json.Marshal(membersValue)
		if err == nil {
			json.Unmarshal(membersBytes, &members)
		}
	}

	result, err := s.proxmoxClient.UpdatePool(ctx, poolID, comment, members, delete)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update pool: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Pool updated successfully",
		"poolid":  poolID,
		"result":  result,
	})
}

func (s *Server) deletePool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_pool")

	poolID := request.GetString("poolid", "")
	if poolID == "" {
		return mcp.NewToolResultError("poolid parameter is required"), nil
	}
	if request.GetString("confirm", "") != poolID {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", poolID)), nil
	}

	result, err := s.proxmoxClient.DeletePool(ctx, poolID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete pool: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Pool deleted successfully",
		"poolid":  poolID,
		"result":  result,
	})
}

func (s *Server) getPoolMembers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_pool_members")

	poolID := request.GetString("poolid", "")
	if poolID == "" {
		return mcp.NewToolResultError("poolid parameter is required"), nil
	}

	members, err := s.proxmoxClient.GetPoolMembers(ctx, poolID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get pool members: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Pool members retrieved successfully",
		"poolid":  poolID,
		"members": members,
	})
}

// ============ ADDITIONAL TOOLS (Phase 5) ============

func (s *Server) getStorageQuota(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_storage_quota")

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	quota, err := s.proxmoxClient.GetStorageQuota(ctx, storage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get storage quota: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Storage quota information retrieved successfully",
		"storage": storage,
		"quota":   quota,
	})
}

func (s *Server) uploadBackup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: upload_backup")

	storage := request.GetString("storage", "")
	if storage == "" {
		return mcp.NewToolResultError("storage parameter is required"), nil
	}

	backupID := request.GetString("backup_id", "")
	if backupID == "" {
		return mcp.NewToolResultError("backup_id parameter is required"), nil
	}

	filePath := request.GetString("file_path", "")
	if filePath == "" {
		return mcp.NewToolResultError("file_path parameter is required"), nil
	}

	result, err := s.proxmoxClient.UploadBackup(ctx, storage, backupID, filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to upload backup: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message":   "Backup upload initiated",
		"storage":   storage,
		"backup_id": backupID,
		"result":    result,
	})
}

func (s *Server) getNodeLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_logs")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	lines := request.GetInt("lines", 50)

	logs, err := s.proxmoxClient.GetNodeLogs(ctx, nodeName, lines)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node logs: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node system logs retrieved successfully",
		"node":    nodeName,
		"lines":   lines,
		"logs":    logs,
	})
}

func (s *Server) getNodeAPTUpdates(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_apt_updates")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	updates, err := s.proxmoxClient.GetNodeAPTUpdates(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get APT updates: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Available package updates retrieved successfully",
		"node":    nodeName,
		"updates": updates,
	})
}

func (s *Server) applyNodeUpdates(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: apply_node_updates")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	result, err := s.proxmoxClient.ApplyNodeUpdates(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to apply node updates: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "System updates installation initiated",
		"node":    nodeName,
		"result":  result,
	})
}

func (s *Server) getNodeNetwork(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_network")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	network, err := s.proxmoxClient.GetNodeNetwork(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node network configuration: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node network configuration retrieved successfully",
		"node":    nodeName,
		"network": network,
	})
}

func (s *Server) getNodeDNS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_node_dns")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	dns, err := s.proxmoxClient.GetNodeDNS(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get node DNS configuration: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node DNS configuration retrieved successfully",
		"node":    nodeName,
		"dns":     dns,
	})
}

// ============ HA (HIGH AVAILABILITY) CLUSTER MANAGEMENT HANDLERS ============

func (s *Server) getHAStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_ha_status")

	status, err := s.proxmoxClient.GetHAStatus(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get HA status: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "HA status retrieved successfully",
		"status":  status,
	})
}

func (s *Server) enableHAResource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: enable_ha_resource")

	sid := request.GetString("sid", "")
	if sid == "" {
		return mcp.NewToolResultError("sid parameter is required"), nil
	}

	comment := request.GetString("comment", "")
	state := request.GetString("state", "")

	result, err := s.proxmoxClient.EnableHAResource(ctx, sid, comment, state)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to enable HA resource: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "HA resource enabled successfully",
		"sid":     sid,
		"result":  result,
	})
}

func (s *Server) disableHAResource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: disable_ha_resource")

	sid := request.GetString("sid", "")
	if sid == "" {
		return mcp.NewToolResultError("sid parameter is required"), nil
	}

	result, err := s.proxmoxClient.DisableHAResource(ctx, sid)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to disable HA resource: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "HA resource disabled successfully",
		"sid":     sid,
		"result":  result,
	})
}

// ============ CLUSTER OPERATIONS HANDLERS ============

func (s *Server) getClusterConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_cluster_config")

	config, err := s.proxmoxClient.GetClusterConfig(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get cluster config: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Cluster configuration retrieved successfully",
		"config":  config,
	})
}

func (s *Server) getClusterNodesStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_cluster_nodes_status")

	status, err := s.proxmoxClient.GetClusterNodesStatus(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get cluster nodes status: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Cluster nodes status retrieved successfully",
		"status":  status,
	})
}

func (s *Server) addNodeToCluster(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: add_node_to_cluster")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	clusterName := request.GetString("cluster_name", "")
	clusterNetwork := request.GetString("cluster_network", "")

	result, err := s.proxmoxClient.AddNodeToCluster(ctx, nodeName, clusterName, clusterNetwork)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add node to cluster: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node added to cluster successfully",
		"node":    nodeName,
		"result":  result,
	})
}

func (s *Server) removeNodeFromCluster(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: remove_node_from_cluster")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	result, err := s.proxmoxClient.RemoveNodeFromCluster(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to remove node from cluster: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Node removed from cluster successfully",
		"node":    nodeName,
		"result":  result,
	})
}

// ============ FIREWALL & NETWORK MANAGEMENT HANDLERS ============

func (s *Server) getFirewallRules(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_firewall_rules")

	rules, err := s.proxmoxClient.GetFirewallRules(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get firewall rules: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Firewall rules retrieved successfully",
		"rules":   rules,
		"count":   len(rules),
	})
}

func (s *Server) createFirewallRule(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_firewall_rule")

	direction := request.GetString("direction", "")
	if direction == "" {
		return mcp.NewToolResultError("direction parameter is required"), nil
	}

	action := request.GetString("action", "")
	if action == "" {
		return mcp.NewToolResultError("action parameter is required"), nil
	}

	rule := proxmox.FirewallRule{
		Direction: direction,
		Action:    action,
		Source:    request.GetString("source", ""),
		Dest:      request.GetString("dest", ""),
		Proto:     request.GetString("proto", ""),
		Sport:     request.GetString("sport", ""),
		Dport:     request.GetString("dport", ""),
		Comment:   request.GetString("comment", ""),
		Enable:    request.GetInt("enable", 1),
	}

	if err := s.proxmoxClient.CreateFirewallRule(ctx, rule); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create firewall rule: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Firewall rule created successfully",
		"rule":    rule,
	})
}

func (s *Server) deleteFirewallRule(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: delete_firewall_rule")

	position := request.GetString("position", "")
	if position == "" {
		return mcp.NewToolResultError("position parameter is required"), nil
	}
	if request.GetString("confirm", "") != position {
		return mcp.NewToolResultError(fmt.Sprintf("Delete requires confirmation. Set confirm to %q to proceed.", position)), nil
	}

	if err := s.proxmoxClient.DeleteFirewallRule(ctx, position); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete firewall rule: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message":  "Firewall rule deleted successfully",
		"position": position,
	})
}

func (s *Server) getSecurityGroups(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_security_groups")

	groups, err := s.proxmoxClient.GetSecurityGroups(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get security groups: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Security groups retrieved successfully",
		"groups":  groups,
		"count":   len(groups),
	})
}

func (s *Server) createSecurityGroup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: create_security_group")

	name := request.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name parameter is required"), nil
	}

	group := proxmox.SecurityGroup{
		Name:    name,
		Comment: request.GetString("comment", ""),
	}

	if err := s.proxmoxClient.CreateSecurityGroup(ctx, group); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create security group: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "Security group created successfully",
		"group":   group,
	})
}

func (s *Server) getNetworkInterfaces(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_network_interfaces")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	interfaces, err := s.proxmoxClient.GetNetworkInterfaces(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get network interfaces: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message":    "Network interfaces retrieved successfully",
		"node":       nodeName,
		"interfaces": interfaces,
		"count":      len(interfaces),
	})
}

func (s *Server) getVLANConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.logger.Debug("Tool called: get_vlan_config")

	nodeName := request.GetString("node_name", "")
	if nodeName == "" {
		return mcp.NewToolResultError("node_name parameter is required"), nil
	}

	vlans, err := s.proxmoxClient.GetVLANConfig(ctx, nodeName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get VLAN configuration: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"message": "VLAN configuration retrieved successfully",
		"node":    nodeName,
		"vlans":   vlans,
		"count":   len(vlans),
	})
}
