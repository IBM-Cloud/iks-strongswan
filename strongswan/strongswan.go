/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/IBM-Cloud/iks-strongswan/calico"
	"github.com/IBM-Cloud/iks-strongswan/kube"
	"github.com/IBM-Cloud/iks-strongswan/monitoring"
	"github.com/IBM-Cloud/iks-strongswan/network"
	"github.com/IBM-Cloud/iks-strongswan/utils"
)

// Various constants
const (
	envVarConnectUsingVip  = "CONNECT_USING_LB_IP"
	envVarEnableMonitoring = "ENABLE_MONITORING"
	envVarEnablePodSNAT    = "ENABLE_POD_SNAT"
	envVarEnableSingleIP   = "ENABLE_SINGLE_IP"
	envVarLoadBalancerIP   = "LOAD_BALANCER_IP"
	envVarLocalZoneSubnet  = "LOCAL_ZONE_SUBNET"
	envVarServiceName      = "SERVICE_NAME"
	envVarZoneLoadBalancer = "ZONE_LOAD_BALANCER"

	ipsecConfigDir    = "/etc/ipsec.config/"
	ipsecEtcDir       = "/etc/"
	strongswanEtcDir  = "/etc/strongswan.d/"
	ipsecConf         = "ipsec.conf"
	ipsecSecrets      = "ipsec.secrets"
	strongswanConf    = "strongswan.conf"
	charonloggingConf = "charon-logging.conf"
)

var cleanupCalico []string
var connectUsingVip bool
var enablePodSNAT string
var enableSingleIP bool
var ipsecAuto string
var leftID string
var leftSubnet string
var loadBalancerIP string
var localZoneSubnet string
var monitoringEnabled bool
var remoteGateway string
var requestedLoadBalancerIP string
var rightSubnet string
var serviceName string
var tcpListener *net.TCPListener
var zoneLoadBalancer string

var establishedMap = map[string]string{} // Keep track of IKE_SA connections

// Parse the console output from the strongswan service.  This routine is not really needed anymore since we only log information here
func parseStrongswanOutput(line string) {
	if strings.Contains(line, "state change: CONNECTING => ESTABLISHED") {
		words := strings.Fields(line)
		establishedMap[words[2]] = time.Now().Format("01/02_15:04:05") // Similar format that is shown in log (no year or spaces)
		log.Printf("ESTABLISHED: %v", establishedMap)
		if monitoringEnabled && len(establishedMap) == 1 {
			log.Print("Monitoring of vpn tunnel has started")
			monitoring.Start()
		}
	}
	if strings.Contains(line, "state change: ESTABLISHED => DELETING") || strings.Contains(line, "state change: REKEYING => DELETING") {
		words := strings.Fields(line)
		delete(establishedMap, words[2])
		log.Printf("ESTABLISHED: %v", establishedMap)
		if monitoringEnabled && len(establishedMap) == 0 {
			log.Print("Monitoring of vpn tunnel has stopped")
			monitoring.Stop()
		}
	}
}

// Process the LOCAL_ZONE_SUBNET setting based on which node that VPN pod landed on
func processLocalZoneSubnet(zone string) string {
	subnet := ""
	for _, zoneSubnet := range strings.Split(localZoneSubnet, ";") {
		zoneSplit := strings.Split(zoneSubnet, "=")
		if len(zoneSplit) != 2 {
			log.Fatalf("ERROR: The local.zoneSubnet option is not in format zone=CIDR: %s", zoneSubnet)
		}
		// Make sure the new leftsubnet value is a valid CIDR
		for _, sn := range strings.Split(zoneSplit[1], ",") {
			if _, _, err := net.ParseCIDR(sn); err != nil {
				log.Fatalf("ERROR: Invalid subnet specified in the local.zoneSubnet option: %s", zoneSubnet)
			}
		}
		if zoneSplit[0] == zone {
			subnet = zoneSplit[1]
			break
		}
	}
	if subnet == "" {
		log.Fatalf("ERROR: The worker node zone %s was not specified in the local.zoneSubnet option: %s", zone, localZoneSubnet)
	}
	return subnet
}

// Process the ZONE_LOAD_BALANCER setting based on which node that VPN pod landed on
func processZoneLoadBalancer(zone string) (string, string) {
	loadBalancer := ""
	for _, zoneLb := range strings.Split(zoneLoadBalancer, ",") {
		zoneSplit := strings.Split(zoneLb, "=")
		if len(zoneSplit) != 2 {
			log.Fatalf("ERROR: The zoneLoadBalancer configuration property is not specified correctly: %s", zoneLb)
		}
		if net.ParseIP(zoneSplit[1]) == nil {
			log.Fatalf("ERROR: Invalid IP address specified in the zoneLoadBalancer option: %s", zoneLb)
		}
		if zoneSplit[0] == zone {
			loadBalancer = zoneSplit[1]
		}
	}
	if loadBalancer == "" {
		log.Fatalf("ERROR: The worker node zone %s was not specified in the zoneLoadBalancer configuration property: %s", zone, zoneLoadBalancer)
	}
	return serviceName + "-" + zone, loadBalancer
}

// Perform any initialization needed by the VPN pod
func vpnPodInit() {
	if strings.ToLower(os.Getenv(envVarConnectUsingVip)) == "true" {
		connectUsingVip = true
	}
	if strings.ToLower(os.Getenv(envVarEnableMonitoring)) == "true" {
		monitoringEnabled = true
	}
	if strings.ToLower(os.Getenv(envVarEnableSingleIP)) == "true" {
		enableSingleIP = true
	}
	enablePodSNAT = strings.ToLower(os.Getenv(envVarEnablePodSNAT))
	if enablePodSNAT != "" {
		if enablePodSNAT != "true" && enablePodSNAT != "false" && enablePodSNAT != "auto" {
			log.Fatalf("ERROR: Invalid value specified for enablePodSNAT: %s", enablePodSNAT)
		}
	}
	serviceName = os.Getenv(envVarServiceName)
	if serviceName == "" {
		serviceName = releaseName + "-strongswan"
	}
	requestedLoadBalancerIP = os.Getenv(envVarLoadBalancerIP)
	zoneLoadBalancer = os.Getenv(envVarZoneLoadBalancer)
	localZoneSubnet = os.Getenv(envVarLocalZoneSubnet)

	// Copy the configuration files to the correct locations
	utils.CopyConfigFile(ipsecConf, ipsecConfigDir, ipsecEtcDir, true)
	utils.CopyConfigFile(ipsecSecrets, ipsecConfigDir, ipsecEtcDir, true)
	utils.CopyConfigFile(strongswanConf, ipsecConfigDir, ipsecEtcDir, false)
	utils.CopyConfigFile(charonloggingConf, ipsecConfigDir, strongswanEtcDir, false)

	// Validate contents of ipsec.conf
	configData := utils.ValidateConfig(filepath.Join(ipsecEtcDir, ipsecConf))
	leftID = utils.ExtractConfigData(configData, utils.ConfigDataLeftID)
	leftSubnet = utils.ExtractConfigData(configData, utils.ConfigDataLeftSubnet)
	rightSubnet = utils.ExtractConfigData(configData, utils.ConfigDataRightSubnet)
	remoteGateway = utils.ExtractConfigData(configData, utils.ConfigDataRemoteGateway)
	ipsecAuto = utils.ExtractConfigData(configData, utils.ConfigDataIpsecAuto)
	if ipsecAuto == "add" {
		connectUsingVip = false
	}
}

// Perform initial configuration of the VPN pod
func vpnPodConfig(kubectl *kubernetes.Clientset) {
	if disableRouting {
		return
	}

	// Get the pod name
	log.Print("Set up routing through the VPN pod")
	vpnPodName := os.Getenv(envVarPodName)
	if vpnPodName == "" {
		log.Fatalf("ERROR: Required environment variable %s was not specified", envVarPodName)
	}
	log.Printf("   vpn pod name: %v", vpnPodName)

	vpnPodIP, workerNodeIP := kube.GetPodInfo(kubectl, namespace, vpnPodName)
	log.Printf("   vpn pod ip: %v", vpnPodIP)
	log.Printf("   worker node private ip: %v", workerNodeIP)
	validateIPNotInRemoteSubnet(vpnPodIP, rightSubnet)
	validateIPNotInRemoteSubnet(workerNodeIP, rightSubnet)

	nodePublicIP := ""
	// Don't bother extracting node public IP unless required
	if leftID == utils.LeftIDNodePublicIP {
		nodePublicIP = kube.GetNodePublicIP(kubectl, workerNodeIP)
		log.Printf("   worker node public ip: %v", nodePublicIP)
	}

	// If ZONE_LOAD_BALANCER was configured, adjust the serviceName and loadBalancer based on worker node zone
	if zoneLoadBalancer != "" || localZoneSubnet != "" {
		zone := kube.GetNodeZone(kubectl, workerNodeIP)
		log.Printf("   worker node zone: %v", zone)
		if zoneLoadBalancer != "" {
			serviceName, requestedLoadBalancerIP = processZoneLoadBalancer(zone)
		}
		if localZoneSubnet != "" {
			localZoneSubnet = processLocalZoneSubnet(zone)
		}
	}

	loadBalancerIP = kube.GetLoadBalancerIP(kubectl, namespace, serviceName, requestedLoadBalancerIP, ipsecAuto, connectUsingVip)
	log.Printf("   load balancer ip: %v", loadBalancerIP)
	if loadBalancerIP == "<pending>" {
		connectUsingVip = false
	}
	routeTable := kube.CalculateRouterTable(loadBalancerIP)
	log.Printf("   route table: %v", routeTable)

	// Update the ipsec.conf leftid and leftsubnet values if necessary
	configFile := filepath.Join(ipsecEtcDir, ipsecConf)
	utils.UpdateConfigLeftID(configFile, leftID, nodePublicIP, loadBalancerIP)
	utils.UpdateConfigLeftSubnet(configFile, leftSubnet, localZoneSubnet)
	if leftSubnet == utils.LeftSubnetZoneSpecific {
		leftSubnet = localZoneSubnet
	}

	workerSubnet := calico.GetNodeSubnet(workerNodeIP)
	log.Printf("   worker subnet: %v", workerSubnet)

	vpnPodDevice := calico.GetPodInterface()
	log.Printf("   vpn pod device name: %v", vpnPodDevice)

	// Initialize the monitoring logic if enabled
	if monitoringEnabled {
		clusterID := kube.GetClusterID(kubectl)
		monitoring.Init(vpnPodName, clusterID)
	}

	// Apply subnet NAT tables inside of the VPN pod
	if localSubnetNAT != "" {
		network.ConfigureSubnetNAT(localSubnetNAT, rightSubnet)
	} else if enableSingleIP {
		network.ConfigureSingleSourceIP(leftSubnet, rightSubnet)
	}
	if remoteSubnetNAT != "" {
		origSubnets := leftSubnet
		additionalEntries := ""
		// When using localSubnetNAT the leftSubnet field is configured with post-translation addresses
		if localSubnetNAT != "" {
			// We need to translate these back to the original addresses in order to get the right rules
			for _, rule := range strings.Split(localSubnetNAT, ",") {
				orig := strings.Split(rule, "=")[0]
				mapped := strings.Split(rule, "=")[1]
				// If the NAT rule specifies an exact match to a defined subnet, replace it
				// Otherwise we add an additional (virtual leftSubnet) entry for our NAT rules
				if strings.Contains(leftSubnet, mapped) {
					origSubnets = strings.ReplaceAll(leftSubnet, mapped, orig)
				} else {
					additionalEntries += orig + ","
				}
			}
		} else if enableSingleIP {
			// If enableSingleIP is enabled, then the "untranslated" leftSubnet is "any traffic"
			origSubnets = "0.0.0.0/0"
		}
		network.ConfigureSubnetNAT(remoteSubnetNAT, additionalEntries+origSubnets)
	}

	// If "auto" was specified for enablePodSNAT, then translate it to true/false now that we know the vpnPodIP
	if enablePodSNAT == "auto" {
		enablePodSNAT = strconv.FormatBool(!network.IsAddrInSubnet(vpnPodIP, leftSubnet))
	}

	// Create/update the calico IPPool for each remote subnet
	if enablePodSNAT == "false" {
		poolSubnets := rightSubnet
		// If there is NAT, we need to translate the subnets before we create the pools
		if remoteSubnetNAT != "" {
			for _, rule := range strings.Split(remoteSubnetNAT, ",") {
				orig := strings.Split(rule, "=")[0]
				mapped := strings.Split(rule, "=")[1]
				poolSubnets = strings.ReplaceAll(poolSubnets, orig, mapped)
			}
		}

		// Create the required IPPools
		for _, subnet := range strings.Split(poolSubnets, ",") {
			log.Printf("   creating IPPool for subnet: %v", subnet)
			calico.CreateIPPool(subnet)
			cleanupCalico = append(cleanupCalico, subnet)
		}
	}

	// If we are forcing outbound traffic through the LoadBalancer VIP
	if connectUsingVip {
		log.Printf("   creating IPPool for remote gateway: %v", remoteGateway)
		calico.CreateIPPool(remoteGateway + "/32")
		cleanupCalico = append(cleanupCalico, remoteGateway+"/32")

		log.Print("   creating a TCP listener so that route daemon can inform us when SNAT rule is in place")
		addr := net.TCPAddr{Port: 4500}
		var err error
		tcpListener, err = net.ListenTCP("tcp", &addr)
		if err != nil {
			log.Fatalf("ERROR: Failed to create listener: %v", err)
		}
	}

	// Update the config map
	configMapData := kube.RouteData{
		ConnectUsingLB: strconv.FormatBool(connectUsingVip),
		LoadBalancerIP: loadBalancerIP,
		LocalSubnet:    leftSubnet,
		RemoteGateway:  remoteGateway,
		RemoteSubnet:   rightSubnet,
		RouteTable:     routeTable,
		VpnPodDevice:   vpnPodDevice,
		VpnPodIP:       vpnPodIP,
		VpnPodName:     vpnPodName,
		WorkerNodeIP:   workerNodeIP,
		WorkerSubnet:   workerSubnet,
	}
	log.Printf("   updating config map: %v", configMapName)
	kube.UpdateConfigMap(kubectl, namespace, configMapName, configMapData)
}

// Perform any cleanup necessary of the VPN pod
func vpnPodWaitRouteDaemon() {
	if !connectUsingVip {
		return
	}

	// Wait for route daemon to connect to us
	log.Print("Wait for the route daemon set to apply iptable rules on this worker node")
	conn, err := tcpListener.AcceptTCP()
	if err != nil {
		log.Fatalf("ERROR: Failed to wait for incoming connection: %v", err)
	}

	// Close done the incoming connection and the listener
	log.Print("Incoming connection received.  Continue with VPN pod start up logic")
	conn.Close()        // #nosec G104 ok to ignore error on close
	tcpListener.Close() // #nosec G104 ok to ignore error on close
}

// Perform any cleanup necessary of the VPN pod
func vpnPodCleanup() {
	if len(cleanupCalico) > 0 {
		log.Print("Clean up resources allocated in calico")
		for _, subnet := range cleanupCalico {
			log.Printf("   deleting IPPool for subnet: %v", subnet)
			calico.DeleteIPPool(subnet)
		}
		cleanupCalico = []string{}
		log.Print("Successfully cleaned up calico")
	}
	if monitoringEnabled {
		log.Print("Cancel the monitor thread")
		monitoring.Cancel()
		monitoringEnabled = false
	}
}
