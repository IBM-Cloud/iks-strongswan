/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2020 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package kube

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var serviceRetryCount = 5

// CalculateRouterTable - Determine which router table to use based on load balancer IP
func CalculateRouterTable(loadBalancerIP string) string {
	// If external IP was not assigned to load balancer, default to table 200
	if loadBalancerIP == "<pending>" {
		return "200"
	}

	// Since we are only looking at the last byte of the public IP subnet, this should result in
	// values of 202-205 and 210-213 assuming that /29 is used for public range
	lastDigit := strings.Split(loadBalancerIP, ".")[3]
	num, err := strconv.Atoi(lastDigit)
	if err != nil {
		log.Printf("strconv.Atoi failed: %v", err)
		num = 0
	}

	// Return 200 + [1 ... 15]
	return strconv.Itoa(200 + (num & 0xF))
}

// GetLoadBalancerIP - Get the Load Balancer IP for a specific service
func GetLoadBalancerIP(client *kubernetes.Clientset, namespace, serviceName, requestedLoadBalancerIP, ipsecAuto string, connectUsingVip bool) string {
	for i := 1; i <= serviceRetryCount; i++ {
		service, err := client.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("ERROR: Failed to get service: %v", err)
		}

		// If external IP was assigned to the service, return with that IP address
		if len(service.Status.LoadBalancer.Ingress) > 0 {
			return service.Status.LoadBalancer.Ingress[0].IP
		}

		// Don't wait for an external IP to be assigned to Kube service if (1) outbound connection, (2) LB IP was not specified, AND (3) not using LB IP for connect
		if ipsecAuto == "start" && requestedLoadBalancerIP == "" && !connectUsingVip {
			break
		}

		// Service does not have a LB IP yet. Sleep for a second and try again
		log.Print("   load balancer ip: <pending>")
		time.Sleep(time.Second)
	}

	// If the user requested a specific Load Balancer IP address, fail if it was not assigned to the service
	if requestedLoadBalancerIP != "" {
		log.Fatalf("ERROR: Load balancer service was not assigned the requested external IP: %s", requestedLoadBalancerIP)
	}

	// If setting up a listening VPN service, fail if we didn't get an public IP address
	if ipsecAuto == "add" {
		log.Fatalf("ERROR: Load balancer VPN service was not assigned a public IP")
	}

	// Return string indicating external IP was not assigned to the service
	return "<pending>"
}
