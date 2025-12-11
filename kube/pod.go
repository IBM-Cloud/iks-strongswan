/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2020 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package kube provides GO methods for Kubernetes resources
package kube

import (
	"context"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var podRetryCount = 10

// GetPodInfo - Get the pod IP and worker node IP
func GetPodInfo(client *kubernetes.Clientset, namespace, podName string) (string, string) {
	for i := 1; i <= podRetryCount; i++ {
		pod, err := client.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("ERROR: Failed to locate pod: %v", err)
		}
		// Verify the pod is in "Running" state.  After 10 tries, ignore the state
		if pod.Status.Phase == corev1.PodRunning || i == podRetryCount {
			return pod.Status.PodIP, pod.Status.HostIP
		}
		// Pod is not active yet. Sleep for a second and try again
		log.Printf("   pod <%s> has status: %v", podName, pod.Status.Phase)
		time.Sleep(time.Second)
	}
	// Should never get here.  On last loop, we return with first pod found (regardless of the state)
	return "", ""
}
