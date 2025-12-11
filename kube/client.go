/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Bluemix Container Service, 5737-D43
 * (C) Copyright IBM Corp. 2017 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package kube

import (
	"log"
	"os"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // Needed for armada
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClient - Create client connection to kubernetes master
func GetClient() *kubernetes.Clientset {
	var config *rest.Config
	var err error

	// Create the client config
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	}
	if err != nil {
		log.Fatalf("ERROR: Failed to get config: %v", err)
		return nil
	}

	// Create the client
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("ERROR: Failed to get client: %v", err)
	}
	return client
}
