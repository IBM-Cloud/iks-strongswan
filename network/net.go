/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2025 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package network provides GO methods for Linux network functions
package network

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
)

// NetAddDelAction - What type of action should be done by Add/Remove routines if the operation fails
type NetAddDelAction string

const (
	// NetActionAdd - Flag to indicate routes/rules should be added
	NetActionAdd NetAddDelAction = "add"

	// NetActionDelete - Flag to indicate routes/rules should be deleted
	NetActionDelete NetAddDelAction = "del"
)

// RoutingInfo - only need to store destination, via gateway, and dev interface being used
type RoutingInfo struct {
	Dest, Via, Dev string
}

// ConfigureSubnetNAT - Configure a local or remote NAT table
// rulesNAT - The localSubnetNAT or remoteSubnetNAT rules to be applied
// subnetList -
// The list of subnets that will be "subject" to the rules. ie, the subnets that will SEE the remapped IPs
// Typically this is the "rightSubnet" list when applying localSubnetNAT rules, and the "leftSubnet"
// when applying remoteSubnetNAT rules
func ConfigureSubnetNAT(rulesNAT, subnetList string) {
	for _, subject := range strings.Split(subnetList, ",") {
		for _, rule := range strings.Split(rulesNAT, ",") {
			ruleSplit := strings.Split(rule, "=")
			original := ruleSplit[0]
			mapped := ruleSplit[1]
			if !strings.HasSuffix(original, "/32") && strings.HasSuffix(mapped, "/32") {
				singleIP := strings.Split(mapped, "/")[0]
				command := fmt.Sprintf("-t nat -A POSTROUTING -s %s -d %s -j SNAT --to %s", original, subject, singleIP)
				ipTablesRun(command)
			} else {
				command := fmt.Sprintf("-t nat -A POSTROUTING -s %s -d %s -j NETMAP --to %s", original, subject, mapped)
				ipTablesRun(command)
				command = fmt.Sprintf("-t nat -A PREROUTING -s %s -d %s -j NETMAP --to %s", subject, mapped, original)
				ipTablesRun(command)
			}
		}
	}
}

// ConfigureSingleSourceIP - Configure single source IP
func ConfigureSingleSourceIP(localSubnet, remoteSubnet string) {
	if !strings.HasSuffix(localSubnet, "/32") || strings.Contains(localSubnet, ",") {
		log.Printf("WARNING: The configuration option local.subnet: %s is not a single /32 subnet.  Single source IP is not enabled", localSubnet)
		return
	}
	singleIP := strings.Split(localSubnet, "/")[0]
	command := fmt.Sprintf("-t nat -A POSTROUTING -d %s -j SNAT --to %s", remoteSubnet, singleIP)
	ipTablesRun(command)
}

// ConfigureSNAT - Configure SNAT rules on worker nodes
func ConfigureSNAT(addDelAction NetAddDelAction, remoteGateway, vpnPodIP, localBalancerIP string) {
	action := "A"
	if addDelAction == NetActionDelete {
		action = "D"
	}
	command := ""
	if vpnPodIP != "" {
		command = fmt.Sprintf("-t nat -%s POSTROUTING -d %s -s %s -p udp -j SNAT --to %s", action, remoteGateway, vpnPodIP, localBalancerIP)
	} else {
		command = fmt.Sprintf("-t nat -%s POSTROUTING -d %s -j MASQUERADE", action, remoteGateway)
	}
	ipTablesRun(command)
}

// DeleteConntrackEntry - Delete stale conntrack entry
func DeleteConntrackEntry(remoteGateway, localBalancerIP string) {
	command := fmt.Sprintf("/usr/sbin/conntrack -D -s %s -d %s -p udp", remoteGateway, localBalancerIP)
	log.Printf("%s", command)
	words := strings.Fields(command)
	outBytes, _ := exec.Command("sudo", words...).CombinedOutput() // #nosec G104,G204 variable is built from fixed constants and network information, user can not override, ok to ignore error
	outArray := strings.Split(string(outBytes), "\n")
	for _, line := range outArray {
		if len(line) > 1 {
			log.Printf("%s", line)
		}
	}
}

// FlushLocalSubnetNAT - Flush the settings of the local subnet NAT table
func FlushLocalSubnetNAT() {
	ipTablesRun("--flush -t nat")
}

// GetDeviceToWorkerNode - Get the device to route data over to get to worker node
func GetDeviceToWorkerNode(workerNode string) string {
	device := ""
	outBytes, err := exec.Command("sudo", "/sbin/ip", "route", "list").CombinedOutput()
	if err != nil {
		log.Fatalf("ERROR: Failed to retrieve routing table: %v", err)
	}
	outArray := strings.Split(string(outBytes), "\n")
	for _, line := range outArray {
		if strings.Contains(line, workerNode) {
			words := strings.Fields(line)
			for i, word := range words {
				if word == "dev" && i < len(words)-1 {
					device = words[i+1]
				}
			}
		}
	}
	// If we did not find a route to the VPN pod worker node, calico-node probably has not added it yet
	// We can't do anything if we don't have a device name, therefore exit with a fatal error
	if device == "" {
		log.Print("Current routes on the node:")
		for _, line := range outArray {
			if len(line) > 0 {
				log.Printf("\t%s", line)
			}
		}
		log.Fatalf("ERROR: Unable to find route to VPN pod worker node: %s", workerNode)
	}
	return device
}

// GetRoutingTable - Retrieve the routing table and return it in an array of RoutingInfo object.
func GetRoutingTable() []RoutingInfo {
	var routes []RoutingInfo
	outBytes, err := exec.Command("sudo", "/sbin/ip", "route").CombinedOutput()
	if err != nil {
		log.Printf("ERROR: Failed to retrieve routing table: %v", err)
		return routes
	}
	log.Print("Routing Table:")
	outArray := strings.Split(string(outBytes), "\n")
	for _, entry := range outArray {
		words := strings.Fields(entry)
		if len(words) >= 3 {
			log.Printf("\t%s", entry)
			var route = RoutingInfo{}
			route.Dest = words[0]
			if route.Dest != "default" && !strings.Contains(route.Dest, "/") {
				route.Dest += "/32"
			}
			for n, value := range words {
				if value == "via" && n < len(words)-1 {
					route.Via = words[n+1]
				} else if value == "dev" && n < len(words)-1 {
					route.Dev = words[n+1]
				}
			}
			routes = append(routes, route)
		}
	}
	return routes
}

// ipTablesRun - Run iptables command helper routine
func ipTablesRun(command string) {
	log.Printf("iptables-legacy %s", command)
	ipTablesCommand := fmt.Sprintf("/usr/sbin/iptables-legacy %v", command)
	words := strings.Fields(ipTablesCommand)
	err := exec.Command("sudo", words...).Run() // #nosec G204 variable is built from fixed constants and network information, user can not override
	if err != nil {
		log.Printf("WARNING: Failed <iptables-legacy %s>: %v", command, err)
	}
}

// IsAddrInSubnet - Is the IP address in one of the subnets provided
func IsAddrInSubnet(ipAddr, subnets string) bool {
	ip := net.ParseIP(ipAddr)
	if ip != nil {
		for _, subnet := range strings.Split(subnets, ",") {
			_, ipNet, err := net.ParseCIDR(subnet)
			if err == nil && ipNet.Contains(ip) {
				return true
			}
		}
	}
	return false
}

// IsAddrLocal - If the specified IP address located on this host
func IsAddrLocal(ipAddr string) bool {
	addrList, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("ERROR: Failed to retrieve list of local IP addresses: %v", err)
		return false
	}
	for _, addr := range addrList {
		if strings.HasPrefix(addr.String(), ipAddr) {
			return true
		}
	}
	return false
}

// ListRoutes - List the current route for a specific table
func ListRoutes(routeTable string) {
	log.Printf("ip route list table %s", routeTable)
	outBytes, err := exec.Command("sudo", "/sbin/ip", "route", "list", "table", routeTable).CombinedOutput() // #nosec G204 variable is built from fixed constants and network information, user can not override
	if err != nil {
		log.Printf("ERROR: Failed to retrieve routing table %s: %v", routeTable, err)
		return
	}
	outArray := strings.Split(string(outBytes), "\n")
	for _, line := range outArray {
		if len(line) > 1 {
			log.Printf("\t%s", line)
		}
	}
}

// ListRules - List the current ip rules
func ListRules() {
	log.Print("ip rules list")
	outBytes, err := exec.Command("sudo", "/sbin/ip", "rule", "list").CombinedOutput()
	if err != nil {
		log.Printf("ERROR: Failed to retrieve rule list: %v", err)
		return
	}
	outArray := strings.Split(string(outBytes), "\n")
	for _, line := range outArray {
		if strings.Contains(line, ":") {
			log.Printf("\t%s", line)
		}
	}
}

// RouteRemoteSubnet - add/del routing for 1 or more remote subnets
func RouteRemoteSubnet(addDelAction NetAddDelAction, remoteSubnet, routeInfo string) {
	for _, subnet := range strings.Split(remoteSubnet, ",") {
		_, networkAddr, err := net.ParseCIDR(subnet)
		if err != nil {
			continue
		}
		UpdateRoute(addDelAction, networkAddr.String(), routeInfo)
	}
}

// TunnelNeededToReachSubnet - Is a tunnel needed to reach this local subnet (based on routing table that is passed in)
func TunnelNeededToReachSubnet(localSubnet string, routingTable []RoutingInfo) bool {
	_, ipNet, err := net.ParseCIDR(localSubnet)
	if err != nil {
		return false
	}
	for _, route := range routingTable {
		if route.Dev == "tunl0" { // Found a route that requires a tunnel
			if ipNet.Contains(net.ParseIP(route.Via)) { // Is the gateway for this route in the subnet that was passed in
				return true
			}
		}
	}
	return false
}

// UpdateRoute - add/del routing info for a subnet
func UpdateRoute(addDelAction NetAddDelAction, subnet, routeInfo string) {
	routeCommand := fmt.Sprintf("/sbin/ip route %s %s %s", addDelAction, subnet, routeInfo)
	log.Printf("%s", routeCommand)
	words := strings.Fields(routeCommand)
	err := exec.Command("sudo", words...).Run() // #nosec G204 variable is built from fixed constants and network information, user can not override
	if err != nil {
		log.Printf("WARNING: Failed <%s>: %v", routeCommand, err)
	}
}

// UpdateRouteRule - Update the route rules (add/del) as needed depending on if they already exist
func UpdateRouteRule(addDelAction NetAddDelAction, fromSource, routeTable string) {
	fromSource = strings.TrimSuffix(fromSource, "/32")
	outBytes, err := exec.Command("sudo", "/sbin/ip", "rule", "list").CombinedOutput()
	if err != nil {
		log.Printf("ERROR: Failed to retrieve routing rules: %v", err)
		return
	}
	foundRule := false
	outArray := strings.Split(string(outBytes), "\n")
	for _, line := range outArray {
		if strings.HasPrefix(line, routeTable+":") && strings.Contains(line, "from "+fromSource) {
			foundRule = true
		}
	}
	routesExist := false
	if addDelAction == NetActionDelete {
		outBytes, err := exec.Command("sudo", "/sbin/ip", "route", "list", "table", routeTable).CombinedOutput() // #nosec G204 variable is built from fixed constants and network information, user can not override
		if err != nil {
			log.Printf("ERROR: Failed to retrieve routing table %s: %v", routeTable, err)
			return
		}
		outArray := strings.Split(string(outBytes), "\n")
		for _, line := range outArray {
			if len(line) > 1 {
				routesExist = true
			}
		}
	}
	var ruleCommand string

	// Process "add" to the rule table
	if addDelAction == NetActionAdd {
		// If table already exists, no need to add it
		if foundRule {
			log.Printf("Rule for table %s from source %s already exists", routeTable, fromSource)
			return
		}
		ruleCommand = fmt.Sprintf("rule %s from %s table %s prior %s", addDelAction, fromSource, routeTable, routeTable)
	}

	// Process "del" to the rule table
	if addDelAction == NetActionDelete {
		// if table does not exist, no need to remove it
		if !foundRule {
			log.Printf("Rule for table %s from source %s does not exists", routeTable, fromSource)
			return
		}
		if routesExist && fromSource == "all" {
			log.Printf("Rule was not deleted.  Routes are still defined on table %s", routeTable)
			return
		}
		ruleCommand = fmt.Sprintf("rule %s from %s table %s", addDelAction, fromSource, routeTable)
	}

	log.Printf("ip %s", ruleCommand)
	ipRuleCommand := fmt.Sprintf("/sbin/ip %v", ruleCommand)
	words := strings.Fields(ipRuleCommand)
	err = exec.Command("sudo", words...).Run() // #nosec G204 variable is built from fixed constants and network information, user can not override
	if err != nil {
		log.Printf("WARNING: Failed <ip %s>: %v", ruleCommand, err)
	}
}
