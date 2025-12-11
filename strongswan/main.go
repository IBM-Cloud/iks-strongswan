/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2025 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Progran strongswan provides configuration of the strongSwan service
package main

import (
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/IBM-Cloud/iks-strongswan/calico"
	"github.com/IBM-Cloud/iks-strongswan/kube"
	"github.com/IBM-Cloud/iks-strongswan/network"
	"github.com/IBM-Cloud/iks-strongswan/utils"
)

// Various constants
const (
	envVarBuildDate       = "BUILD_DATE"
	envVarDisableRouting  = "DISABLE_ROUTING"
	envVarDisableVpn      = "DISABLE_VPN"
	envVarLocalSubnetNAT  = "LOCAL_SUBNET_NAT"
	envVarNamespace       = "NAMESPACE"
	envVarPodIP           = "POD_IP"
	envVarPodName         = "POD_NAME"
	envVarReleaseName     = "RELEASE_NAME"
	envVarRemoteSubnetNAT = "REMOTE_SUBNET_NAT"
	envVarRouteDaemon     = "ROUTE_DAEMON"
	envVarRunHelmTest     = "RUN_HELM_TEST"

	runHelmCommand = "/usr/local/bin/runHelmTest"
)

var configMapName string
var disableRouting bool
var disableVpn bool
var localSubnetNAT string
var namespace string
var releaseName string
var remoteSubnetNAT string
var routeDaemon bool                          // Are we running in the route daemon ?
var signalReceivedChan = make(chan string, 1) // Channel used by indicate that the signal handler was invoked

var strongswan = utils.ShellCommand{
	Path: "/usr/sbin/",
	Name: "ipsec",
	Args1: []string{
		"start", "--nofork",
	},
	MonitorOutput: true,
	RunAsRoot:     true,
}

// Signal handler for SIGINT / SIGTERM.  In this handler we need to remove the status file
// if we have been updating it and stop the ipsec service
func handleSignal() {
	log.Print("Register signal handler...")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)
	<-signalChan
	log.Print("Signal caught, terminating process...")
	vpnPodCleanup()
	strongswan.Stop()
	handleRoutes(savedRouteMap, network.NetActionDelete)
	signalReceivedChan <- "Signal"
	log.Print("Exiting signal handler")
}

// Initialize global variables and ensure required environment variables were specified
func initialize() {
	helmTest := os.Getenv(envVarRunHelmTest)
	if helmTest != "" {
		runHelmTest(helmTest)
	}
	namespace = os.Getenv(envVarNamespace)
	if namespace == "" {
		log.Fatalf("ERROR: Required environment variable %s was not specified", envVarNamespace)
	}
	releaseName = os.Getenv(envVarReleaseName)
	if releaseName == "" {
		log.Fatalf("ERROR: Required environment variable %s was not specified", envVarReleaseName)
	}
	configMapName = releaseName + "-strongswan-routes"
	localSubnetNAT = strings.ToLower(os.Getenv(envVarLocalSubnetNAT))
	if localSubnetNAT != "" {
		validateNATrules(localSubnetNAT, "local")
	}
	remoteSubnetNAT = strings.ToLower(os.Getenv(envVarRemoteSubnetNAT))
	if remoteSubnetNAT != "" {
		validateNATrules(remoteSubnetNAT, "remote")
	}
	if strings.ToLower(os.Getenv(envVarDisableRouting)) == "true" {
		disableRouting = true
	}
	if strings.ToLower(os.Getenv(envVarDisableVpn)) == "true" {
		disableVpn = true
	}
	if strings.ToLower(os.Getenv(envVarRouteDaemon)) == "true" {
		routeDaemon = true
	}
}

// Invoke the script to run the logic for a given helm test
func runHelmTest(helmTest string) {
	outBytes, err := exec.Command(runHelmCommand, helmTest).CombinedOutput() // #nosec G204 variables used are hard coded compile time constants
	log.Printf("Running helm test: %s\n%v", helmTest, string(outBytes))
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// Validate that IP address is not in the remote subnets
func validateIPNotInRemoteSubnet(ipAddr, remoteSubnet string) {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		log.Fatalf("ERROR: Invalid IP address %v", ipAddr)
	}
	for _, subnet := range strings.Split(remoteSubnet, ",") {
		_, networkAddr, err := net.ParseCIDR(subnet)
		if err != nil {
			log.Fatalf("ERROR: Remote subnet %v is not a valid subnet", subnet)
		}
		if networkAddr != nil && networkAddr.Contains(ip) {
			log.Fatalf("ERROR: Remote subnet %v contains local IP address %v", subnet, ipAddr)
		}
	}
}

func validateNATrules(rules, name string) {
	for _, rule := range strings.Split(rules, ",") {
		ruleSplit := strings.Split(rule, "=")
		if len(ruleSplit) != 2 {
			log.Fatalf("ERROR: The %sSubnetNAT configuration property is not specified correctly: %s", name, rule)
		}
		internal := ruleSplit[0]
		_, intNet, err := net.ParseCIDR(internal)
		if err != nil {
			log.Fatalf("ERROR: Invalid original CIDR specified in %sSubnetNAT: %s", name, internal)
		}
		external := ruleSplit[1]
		_, extNet, err := net.ParseCIDR(external)
		if err != nil {
			log.Fatalf("ERROR: Invalid translated CIDR specified in %sSubnetNAT: %s", name, external)
		}
		if intNet != nil && extNet != nil && intNet.Mask.String() != extNet.Mask.String() && !strings.HasSuffix(external, "/32") {
			log.Fatalf("ERROR: The original/translated CIDR mapping in %sSubnetNAT must be networks of the same size: %s", name, rule)
		}
	}
}

// main routine
func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds) // Display microseconds
	log.Printf("Starting strongswan: %s", os.Getenv(envVarBuildDate))

	// Initialize global vars
	initialize()

	// Register signal handler
	go handleSignal()
	time.Sleep(time.Second)

	// Initialize the calico config and secrets
	var kubectl *kubernetes.Clientset
	if !disableRouting {
		kubectl = kube.GetClient()
		calico.Initialize()
	}

	// Run the route daemon logic if requested
	if routeDaemon {
		// Route daemon specific initialization
		routeDaemonInit()

		// Watch for config map changes
		kube.WatchConfigMap(kubectl, namespace, configMapName, configMapCreated, configMapDeleted, configMapUpdated)
		log.Print("Waiting for config map updates...")
		<-signalReceivedChan
		log.Print("Breaking out of loop")
	} else {
		// VPN pod specific initialization
		vpnPodInit()

		// VPN pod configuration
		vpnPodConfig(kubectl)
		defer vpnPodCleanup()

		// Wait for the route daemon on this node to configure iptable rules
		vpnPodWaitRouteDaemon()

		// Start ipsec and wait for it to end
		if !disableVpn {
			strongswan.Start()
			go strongswan.Monitor(nil, parseStrongswanOutput)
			strongswan.Wait()
		}
	}

	log.Printf("Exiting strongswan: %s", os.Getenv(envVarBuildDate))
}
