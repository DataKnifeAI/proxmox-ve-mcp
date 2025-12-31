package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
)

// FirewallRule represents a firewall rule in Proxmox
type FirewallRule struct {
	ID        string `json:"id,omitempty"`
	Direction string `json:"direction"` // in, out, group
	Action    string `json:"action"`    // ACCEPT, DROP, REJECT
	Source    string `json:"source,omitempty"`
	Dest      string `json:"dest,omitempty"`
	Proto     string `json:"proto,omitempty"`
	Sport     string `json:"sport,omitempty"`
	Dport     string `json:"dport,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Enable    int    `json:"enable"`
}

// SecurityGroup represents a security group/firewall group
type SecurityGroup struct {
	Name    string         `json:"name"`
	Comment string         `json:"comment,omitempty"`
	Rules   []FirewallRule `json:"rules,omitempty"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Iface       string   `json:"iface"`
	Type        string   `json:"type"` // loopback, vlan, veth, tap, etc
	Autostart   int      `json:"autostart,omitempty"`
	Bridge      string   `json:"bridge,omitempty"`
	VLAN        int      `json:"vlan,omitempty"`
	MACAddr     string   `json:"hwaddr,omitempty"`
	IPAddresses []string `json:"ip_addrs,omitempty"`
	MTU         int      `json:"mtu,omitempty"`
}

// VLANConfig represents VLAN configuration
type VLANConfig struct {
	VID       int    `json:"vlan"`
	Name      string `json:"vlan_name,omitempty"`
	Interface string `json:"interface"`
	Comment   string `json:"comment,omitempty"`
}

// GetFirewallRules returns cluster-wide firewall rules
func (c *Client) GetFirewallRules(ctx context.Context) ([]FirewallRule, error) {
	resp, err := c.doRequest(ctx, "GET", "cluster/firewall/rules", nil)
	if err != nil {
		return nil, fmt.Errorf("get firewall rules: %w", err)
	}

	data, ok := resp.([]interface{})
	if !ok {
		return []FirewallRule{}, nil
	}

	var rules []FirewallRule
	for _, item := range data {
		bytes, _ := json.Marshal(item)
		var rule FirewallRule
		if err := json.Unmarshal(bytes, &rule); err == nil {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

// CreateFirewallRule creates a new firewall rule
func (c *Client) CreateFirewallRule(ctx context.Context, rule FirewallRule) error {
	_, err := c.doRequest(ctx, "POST", "cluster/firewall/rules", rule)
	if err != nil {
		return fmt.Errorf("create firewall rule: %w", err)
	}

	return nil
}

// DeleteFirewallRule deletes a firewall rule by position
func (c *Client) DeleteFirewallRule(ctx context.Context, pos string) error {
	path := fmt.Sprintf("cluster/firewall/rules/%s", pos)
	_, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("delete firewall rule: %w", err)
	}

	return nil
}

// GetSecurityGroups returns all security groups (firewall groups)
func (c *Client) GetSecurityGroups(ctx context.Context) ([]SecurityGroup, error) {
	resp, err := c.doRequest(ctx, "GET", "cluster/firewall/groups", nil)
	if err != nil {
		return nil, fmt.Errorf("get security groups: %w", err)
	}

	data, ok := resp.([]interface{})
	if !ok {
		return []SecurityGroup{}, nil
	}

	var groups []SecurityGroup
	for _, item := range data {
		bytes, _ := json.Marshal(item)
		var group SecurityGroup
		if err := json.Unmarshal(bytes, &group); err == nil {
			groups = append(groups, group)
		}
	}

	return groups, nil
}

// CreateSecurityGroup creates a new security group
func (c *Client) CreateSecurityGroup(ctx context.Context, group SecurityGroup) error {
	_, err := c.doRequest(ctx, "POST", "cluster/firewall/groups", group)
	if err != nil {
		return fmt.Errorf("create security group: %w", err)
	}

	return nil
}

// GetNetworkInterfaces returns all network interfaces on a node
func (c *Client) GetNetworkInterfaces(ctx context.Context, nodeName string) (map[string]NetworkInterface, error) {
	path := fmt.Sprintf("nodes/%s/network", nodeName)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("get network interfaces: %w", err)
	}

	data, ok := resp.(map[string]interface{})
	if !ok {
		return map[string]NetworkInterface{}, nil
	}

	interfaces := make(map[string]NetworkInterface)
	for key, item := range data {
		bytes, _ := json.Marshal(item)
		var iface NetworkInterface
		if err := json.Unmarshal(bytes, &iface); err == nil {
			interfaces[key] = iface
		}
	}

	return interfaces, nil
}

// GetVLANConfig returns VLAN configuration
func (c *Client) GetVLANConfig(ctx context.Context, nodeName string) ([]VLANConfig, error) {
	// VLAN info is extracted from network interfaces
	interfaces, err := c.GetNetworkInterfaces(ctx, nodeName)
	if err != nil {
		return nil, err
	}

	var vlans []VLANConfig
	for ifName, iface := range interfaces {
		if iface.Type == "vlan" && iface.VLAN > 0 {
			vlans = append(vlans, VLANConfig{
				VID:       iface.VLAN,
				Name:      ifName,
				Interface: ifName,
				Comment:   fmt.Sprintf("VLAN %d on %s", iface.VLAN, ifName),
			})
		}
	}

	return vlans, nil
}
