/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2020 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package monitoring provides methods for monitoring the VPN connection
package monitoring

import (
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// verifyIP - validate that each IP in slice of string is a valid IP address
func verifyIP(stringToTest []string) {
	for _, v := range stringToTest {
		ip := net.ParseIP(strings.TrimSpace(v))
		if ip == nil {
			log.Fatalf("monitoring | ERROR: monitoring.privateIPs contains an invalid IP address: %s", v)
		}
	}
}

// verifyEndpoint - performs some basic validation of an endpoint. Checks port number, if present, is within acceptable port range, and that the ip of the host can be looked up.
func verifyEndpoint(stringToTest []string) {
	for _, endpoint := range stringToTest {
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = strings.Replace(endpoint, "", "http://", 1) // Doing this in order to use net/url Parse
		}
		u, err := url.Parse(endpoint)
		if err != nil {
			log.Fatalf("monitoring | ERROR: monitoring.httpEndpoints contains an invalid HTTP endpoint: %s, error: %v", endpoint, err)
		}
		// verify port (if present) is valid
		if u.Port() != "" {
			port, err := strconv.Atoi(u.Port())
			if err != nil {
				log.Fatalf("monitoring | ERROR: Invalid HTTP endpoint %s. Converting port number %v to an integer, error: %v", endpoint, u.Port(), err)
			} else if port > 65535 || port < 1 {
				log.Fatalf("monitoring | ERROR: Invalid HTTP endpoint %s. Port number %d not in the range: 1-65535", endpoint, port)
			}
		}
		_, err = net.LookupHost(u.Hostname())
		if err != nil {
			log.Fatalf("monitoring | ERROR: Attempting to resolve HTTP endpoint %v, error: %v", u.Hostname(), err)
		}
	}
}
