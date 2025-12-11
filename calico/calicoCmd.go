/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2022 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package calico provides GO methods for Calico resources
package calico

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

var calicoEnvSh = "/tmp/calicoEnv.sh"

// CreateIPPool - Create calico IPPool resource for the specified subnet
func CreateIPPool(subnet string) {
	outBytes, err := exec.Command("calicoCmd", "createIPPool", subnet).CombinedOutput() // #nosec G204 variable is built from fixed constants and network information, user can not override
	if err != nil {
		details := ""
		_, net, err := net.ParseCIDR(subnet)
		if net != nil && err == nil {
			if net.String() != subnet {
				details = fmt.Sprintf("Invalid subnet. Change config to use: %s. ", net.String())
			}
		}
		//ERROR: Failed to create IPPool for: 10.85.247.249/29. Invalid subnet. Change config to use: 10.85.247.248/29. Error: exit status 1, ErrMsg: Failed to execute command: error with field cidr = ‘10.85.247.249/29’
		log.Fatalf("ERROR: Failed to create IPPool for: %s. %sError: %v, ErrMsg: %s", subnet, details, err, string(outBytes))
	}
	outArray := strings.Split(string(outBytes), "\n")
	for _, line := range outArray {
		if len(line) > 0 {
			log.Printf("%s", line)
		}
	}
}

// DeleteIPPool - Delete calico IPPool resource for the specified subnet
func DeleteIPPool(subnet string) {
	outBytes, err := exec.Command("calicoCmd", "deleteIPPool", subnet).CombinedOutput() // #nosec G204 variable is built from fixed constants and network information, user can not override
	if err != nil {
		log.Fatalf("ERROR: Failed to delete IPPool for %s: %v - %v", subnet, err, string(outBytes))
	}
	outArray := strings.Split(string(outBytes), "\n")
	for _, line := range outArray {
		if len(line) > 0 {
			log.Printf("%s", line)
		}
	}
}

// GetNodeSubnet - Get the subnet for the node IP that was specified
func GetNodeSubnet(workerIP string) string {
	outBytes, err := exec.Command("calicoCmd", "getNodeSubnet", workerIP).CombinedOutput() // #nosec G204 variable is built from fixed constants and network information, user can not override
	nodeIP := strings.TrimSpace(string(outBytes))
	if err != nil {
		log.Fatalf("ERROR: Failed to retrieve node IP for worker node: %v - %v", err, nodeIP)
	}
	_, networkAddr, err := net.ParseCIDR(nodeIP)
	if err != nil {
		log.Fatalf("ERROR: Invalid node IP retrieved from calico: %v - %v", err, nodeIP)
	}
	if networkAddr != nil {
		return networkAddr.String()
	}
	return nodeIP
}

// GetPodInterface - Get the cali* interface name for the current pod (requires pod networking)
func GetPodInterface() string {
	outBytes, err := exec.Command("calicoCmd", "getPodInterface").CombinedOutput()
	podIfc := strings.TrimSpace(string(outBytes))
	if err != nil {
		log.Fatalf("ERROR: Failed to retrieve calico interface: %v - %v", err, podIfc)
	}
	if !strings.HasPrefix(podIfc, "cali") {
		log.Fatalf("ERROR: Invalid interface retrieved from calico: %v", podIfc)
	}
	return podIfc
}

// Initialize - Set environment variables, load certs into files
func Initialize() {
	// Since there are no calicoSecrets (KDD), only need to export one environment variable
	log.Printf("Setting environment variable: DATASTORE_TYPE='kubernetes'")
	calicoEnvBuffer := "#!/bin/sh\n"
	calicoEnvBuffer += "export DATASTORE_TYPE='kubernetes'\n"
	err := os.WriteFile(calicoEnvSh, []byte(calicoEnvBuffer), 0600)
	if err != nil {
		log.Fatalf("ERROR: Failed to write environment settings to %s: %v", calicoEnvSh, err)
	}
}
