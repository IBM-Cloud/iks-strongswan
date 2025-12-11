/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"log"
	"net"
	"os"
	"runtime"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/IBM-Cloud/iks-strongswan/calico"
	"github.com/IBM-Cloud/iks-strongswan/kube"
	"github.com/IBM-Cloud/iks-strongswan/network"
)

// Various constants
const (
	envVarNonClusterSubnet = "NON_CLUSTER_SUBNET"
)

var localIP string
var localSubnet string
var nonClusterSubnet string
var routingTable []network.RoutingInfo // Routing table info for the current node
var routeTunnel bool                   // Is encapsulation needeed to reach another worker node
var savedRouteMap map[string]string    // Used by signal handler to clean up added routes

// Configmap was created.  Add the necessary routes
func configMapCreated(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	log.Printf("ConfigMap created: %v", kube.MapToSortedString(cm.Data))
	handleRoutes(cm.Data, network.NetActionAdd)
	savedRouteMap = cm.Data
}

// Configmap was deleted.  Delete the old routes
func configMapDeleted(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	log.Printf("ConfigMap deleted: %v", kube.MapToSortedString(cm.Data))
	savedRouteMap = nil
	handleRoutes(cm.Data, network.NetActionDelete)
}

// Configmap was updated.  Delete the old routes / Add the new ones.
func configMapUpdated(oldObj, newObj interface{}) {
	oldCm := oldObj.(*corev1.ConfigMap)
	newCm := newObj.(*corev1.ConfigMap)
	if savedRouteMap == nil {
		log.Print("ConfigMap updated - no saved data") // should never occur
		savedRouteMap = oldCm.Data
	}
	savedData := kube.MapToSortedString(savedRouteMap)
	oldData := kube.MapToSortedString(oldCm.Data)
	newData := kube.MapToSortedString(newCm.Data)
	if savedData == newData {
		log.Printf("ConfigMap updated - no change in saved data: %v", newData)
		log.Printf("   old object: %v", oldCm)
		log.Printf("   new object: %v", newCm)
		return
	}
	if savedData != oldData {
		log.Print("ConfigMap updated - saved data does not match old data:") // should never occur
		log.Printf("   saved data: %v", savedData)
		log.Printf("   old data:   %v", oldData)
	}
	log.Printf("ConfigMap updated (old): %v", savedData)
	handleRoutes(savedRouteMap, network.NetActionDelete)
	savedRouteMap = nil

	log.Printf("ConfigMap updated (new): %v", newData)
	handleRoutes(newCm.Data, network.NetActionAdd)
	savedRouteMap = newCm.Data
}

// Handle config map route data.  Single routine will do either ADD or DELETE
func handleRoutes(cmData map[string]string, addDelAction network.NetAddDelAction) {
	routeData := kube.MapToRouteData(cmData)
	if routeData.RemoteSubnet == "" || routeData.RouteTable == "" || routeData.WorkerNodeIP == "" || routeData.WorkerSubnet == "" {
		return
	}

	// Create the list of remote subnets with NAT applied so it can be passed to the routing functions that need it
	remappedRemoteSubnet := routeData.RemoteSubnet
	if remoteSubnetNAT != "" {
		for _, rule := range strings.Split(remoteSubnetNAT, ",") {
			orig := strings.Split(rule, "=")[0]
			mapped := strings.Split(rule, "=")[1]
			remappedRemoteSubnet = strings.ReplaceAll(remappedRemoteSubnet, orig, mapped)
		}
		log.Printf(" - remapped remote subnets based on remoteSubnetNAT: %s", remappedRemoteSubnet)
	}

	log.Printf("Attempting to %s routes/rules", addDelAction)
	if addDelAction == network.NetActionAdd {
		if remoteSubnetNAT == "" {
			validateIPNotInRemoteSubnet(localIP, routeData.RemoteSubnet)
			validateRoutesForRemoteSubnet(routeData.RemoteSubnet)
		} else {
			validateIPNotInRemoteSubnet(localIP, remappedRemoteSubnet)
			validateRoutesForRemoteSubnet(remappedRemoteSubnet)
		}
	}

	deviceName := ""
	if localIP != routeData.WorkerNodeIP {
		deviceName = network.GetDeviceToWorkerNode(routeData.WorkerNodeIP)
	}

	routeInfo := ""
	if localIP == routeData.WorkerNodeIP {
		log.Printf(" - same worker node as the VPN pod: %s", localIP)
		// If there are tunnels in the routing table, we may need to tunnel traffic from on-prem to diff subnet
		if routeTunnel {
			handleRoutesVpnNode(addDelAction, routeData.LocalSubnet, remappedRemoteSubnet)
		}
		if routeData.WorkerNodeIP == routeData.VpnPodIP {
			log.Print("VPN pod is using host networking.  Additional route is not needed")
			return
		}
		routeInfo = "via " + routeData.VpnPodIP + " dev " + routeData.VpnPodDevice + " table " + routeData.RouteTable

		// Add route for nonCluster subnets if requested
		if nonClusterSubnet != "" {
			handleRoutesNonCluster(addDelAction, routeData, nonClusterSubnet)
		}
	} else {
		if localSubnet == routeData.WorkerSubnet {
			log.Printf(" - same subnet as the VPN pod worker node: %s", localSubnet)
		} else {
			log.Printf(" - different subnet than the VPN pod worker node: %s != %s", localSubnet, routeData.WorkerSubnet)
		}

		if localSubnet != routeData.WorkerSubnet || deviceName == "tunl0" {
			routeInfo = "via " + routeData.WorkerNodeIP + " dev " + deviceName + " onlink table " + routeData.RouteTable
		} else {
			routeInfo = "via " + routeData.WorkerNodeIP + " dev " + deviceName + " table " + routeData.RouteTable
		}
	}

	// Update routes / rules and list them out
	network.RouteRemoteSubnet(addDelAction, remappedRemoteSubnet, routeInfo)
	network.UpdateRouteRule(addDelAction, "all", routeData.RouteTable)
	network.ListRules()
	network.ListRoutes(routeData.RouteTable)
	if localIP == routeData.WorkerNodeIP { // VPN worker node
		network.ListRoutes("199")
	}
	if routeData.ConnectUsingLB == "true" {
		handleRoutesSNAT(addDelAction, routeData)
	}
	// If we have a load balancer IP, delete any stale conntrack entry from the remote gateway to the load balancer IP
	if net.ParseIP(routeData.LoadBalancerIP) != nil && net.ParseIP(routeData.RemoteGateway) != nil && runtime.GOOS != "darwin" {
		network.DeleteConntrackEntry(routeData.RemoteGateway, routeData.LoadBalancerIP)
	}
}

// If localNonClusterSubnet was specified in the config map, need to configure iptable rules
func handleRoutesNonCluster(addDelAction network.NetAddDelAction, routeData kube.RouteData, nonClusterSubnet string) {
	// Since calico is configured to not NAT to the local no cluster sunbets, we need to add this rule which masquerades the traffic
	for _, subnet := range strings.Split(nonClusterSubnet, ",") {
		if routeData.WorkerSubnet == subnet {
			log.Printf("WARNING: localNonClusterSubnet specified %s contains the worker node subnet %s", nonClusterSubnet, subnet)
			continue
		}

		if !strings.Contains(routeData.LocalSubnet, subnet) {
			// If the localNonClusterSubnet is not explicitly configured as a local subnet then show a warning but still configure the NAT
			log.Printf("WARNING: localNonClusterSubnet specified but is not configured as a local subnet %v", subnet)
		}
		network.ConfigureSNAT(addDelAction, subnet, "", "")
	}
}

// If connectUsingLB was specified in the config map, need to configure iptable rules
func handleRoutesSNAT(addDelAction network.NetAddDelAction, routeData kube.RouteData) {
	if localIP == routeData.WorkerNodeIP {
		// Special SNAT rules are needed on the worker node where the VPN pod is running
		network.ConfigureSNAT(addDelAction, routeData.RemoteGateway, routeData.VpnPodIP, routeData.LoadBalancerIP)
	}
	// Since calico is configured to not NAT to the remote gateway, we need to add this rule on all worker nodes
	network.ConfigureSNAT(addDelAction, routeData.RemoteGateway, "", "")

	// Finally, if this was as "ADD", we need to incoming the vpnPod that the SNAT rules are in place
	if addDelAction == network.NetActionAdd && localIP == routeData.WorkerNodeIP {
		log.Printf("SNAT rules have been added.  Wake up the vpnPod located here: %v", routeData.VpnPodIP+":4500")
		conn, err := net.Dial("tcp", routeData.VpnPodIP+":4500")
		if err != nil {
			log.Printf("ERROR: Failed to connect to the VPN pod: %v", err)
		}
		if conn != nil {
			conn.Close() // #nosec G104 ok to ignore error on close
		}
	}
}

// If tunnels have been defined, additional routes are needed to handle cross-subnet traffic
func handleRoutesVpnNode(addDelAction network.NetAddDelAction, localSubnetList, remoteSubnetList string) {
	ruleNeeded := false
	for _, localSub := range strings.Split(localSubnetList, ",") { // For each local subnet shared
		if localSub == localSubnet { // If current subnet, no tunnel needed
			continue
		}
		if network.TunnelNeededToReachSubnet(localSub, routingTable) {
			network.UpdateRoute(addDelAction, localSub, "dev tunl0 table 199")
			ruleNeeded = true
		}
	}
	// With the introduction of local subnet NAT, we now need
	// to examine the inside/internal local subnets too
	if localSubnetNAT != "" {
		for _, local := range strings.Split(localSubnetNAT, ",") {
			localSub := strings.Split(local, "=")[0] // Only look at inside network of the NAT
			if localSub == localSubnet {             // If current subnet, no tunnel needed
				continue
			}
			if network.TunnelNeededToReachSubnet(localSub, routingTable) {
				network.UpdateRoute(addDelAction, localSub, "dev tunl0 table 199")
				ruleNeeded = true
			}
		}
	}
	if ruleNeeded {
		for _, remoteSub := range strings.Split(remoteSubnetList, ",") { // For each local subnet shared
			network.UpdateRouteRule(addDelAction, remoteSub, "199") // Always use route table 199 for these tunnel routes
		}
	}
}

// Perform any initialization needed by the route daemon
func routeDaemonInit() {
	routingTable = network.GetRoutingTable()
	for _, route := range routingTable {
		if route.Dev == "tunl0" {
			routeTunnel = true
		}
	}

	// Get the pod IP = worker node private IP
	localIP = os.Getenv(envVarPodIP)
	if localIP == "" {
		log.Fatalf("ERROR: Required environment variable %s was not specified", envVarPodIP)
	}
	log.Printf("local IP: %v", localIP)

	// Determine the local subnet for the current node
	localSubnet = calico.GetNodeSubnet(localIP)
	log.Printf("local subnet: %v", localSubnet)

	// Check to see if non-cluster subnet was configured
	nonClusterSubnet = os.Getenv(envVarNonClusterSubnet)
	if nonClusterSubnet != "" {
		for _, subnet := range strings.Split(nonClusterSubnet, ",") {
			_, _, err := net.ParseCIDR(subnet)
			if err != nil {
				log.Printf("WARNING: local non cluster subnet %v is not a valid subnet", subnet)
				nonClusterSubnet = ""
			}
		}
	}
}

// Validate the remote routes requested vs local routes on the node
func validateRoutesForRemoteSubnet(remoteSubnet string) {
	for _, subnet := range strings.Split(remoteSubnet, ",") {
		for _, route := range routingTable {
			if route.Dest == subnet {
				log.Fatalf("ERROR: Local route %v is already defined for the remote subnet %v", route, subnet)
			}
		}
	}
}
