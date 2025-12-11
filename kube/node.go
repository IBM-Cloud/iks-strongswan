/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2021 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package kube provides GO methods for Kubernetes resources
package kube

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetNodePublicIP - Get the public IP of the specified worker node
func GetNodePublicIP(client *kubernetes.Clientset, nodeIP string) string {

	// Get call searches by node name (assumes node name = node IP, not that way on ICP)
	node, err := client.CoreV1().Nodes().Get(context.TODO(), nodeIP, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("ERROR: Failed to get node: %v", err)
	}

	// Search the addresses for the external IP
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeExternalIP {
			return addr.Address
		}
	}

	// Node does not have an external IP
	return ""
}

// GetNodeZone - Get the zone of the specified worker node
func GetNodeZone(client *kubernetes.Clientset, nodeIP string) string {

	// Get call searches by node name (assumes node name = node IP, not that way on ICP)
	node, err := client.CoreV1().Nodes().Get(context.TODO(), nodeIP, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("ERROR: Failed to get node: %v", err)
	}

	// Search the labels for "ibm-cloud.kubernetes.io/zone"
	zone := node.Labels["ibm-cloud.kubernetes.io/zone"]
	if zone == "" {
		log.Fatalf("ERROR: ibm-cloud.kubernetes.io/zone label is not set for node: %s", nodeIP)
	}
	return zone
}
