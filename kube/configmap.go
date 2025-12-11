/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2024 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package kube provides GO methods for Kubernetes resources
package kube

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// RouteData - routing data that to be stored in the config map
type RouteData struct {
	ConnectUsingLB string // Connect VPN using LB VIP
	LoadBalancerIP string // Load balancer IP address
	LocalSubnet    string // Local subnets to add routes for
	RemoteGateway  string // Remove gateway
	RemoteSubnet   string // Remove subnets to add routes for
	RouteTable     string // Routing table to use (222-235)
	VpnPodDevice   string // VPN pod Interface name
	VpnPodIP       string // VPN pod IP address
	VpnPodName     string // VPN pod name
	WorkerNodeIP   string // Worker node IP address
	WorkerSubnet   string // Worker subnet
}

type clusterInfo struct {
	ClusterID string `json:"cluster_id"`
}

// List of validation types - used when calling validateValueIsCorrect()
const (
	keyConnectUsingLB = "connectUsingLB"
	keyLoadBalancerIP = "loadBalancerIP"
	keyLocalSubnet    = "localSubnet"
	keyRemoteGateway  = "remoteGateway"
	keyRemoteSubnet   = "remoteSubnet"
	keyRouteTable     = "routeTable"
	keyVpnPodDevice   = "vpnPodDevice"
	keyVpnPodIP       = "vpnPodIP"
	keyVpnPodName     = "vpnPodName"
	keyWorkerNodeIP   = "workerNodeIP"
	keyWorkerSubnet   = "workerSubnet"
)

// Retrieve an existing config map
func getConfigMap(client *kubernetes.Clientset, namespace, configMapName string) (*corev1.ConfigMap, error) {
	return client.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
}

// GetClusterID - Retrieve Cluster Id (if it exists) for IKS environments
func GetClusterID(client *kubernetes.Clientset) (clusterID string) {
	configMap, err := getConfigMap(client, "kube-system", "cluster-info")
	if err != nil {
		return ""
	}
	configMapData, ok := configMap.Data["cluster-config.json"]
	if !ok {
		return ""
	}
	clusterData := &clusterInfo{}
	err = json.Unmarshal([]byte(configMapData), clusterData)
	if err != nil {
		return ""
	}
	return clusterData.ClusterID
}

// MapToRouteData - Extract from data map[] and return RouteData struct
func MapToRouteData(mapData map[string]string) RouteData {
	routeData := RouteData{
		ConnectUsingLB: mapData[keyConnectUsingLB],
		LoadBalancerIP: mapData[keyLoadBalancerIP],
		LocalSubnet:    mapData[keyLocalSubnet],
		RemoteGateway:  mapData[keyRemoteGateway],
		RemoteSubnet:   mapData[keyRemoteSubnet],
		RouteTable:     mapData[keyRouteTable],
		VpnPodDevice:   mapData[keyVpnPodDevice],
		VpnPodIP:       mapData[keyVpnPodIP],
		VpnPodName:     mapData[keyVpnPodName],
		WorkerNodeIP:   mapData[keyWorkerNodeIP],
		WorkerSubnet:   mapData[keyWorkerSubnet],
	}
	return routeData
}

// MapToSortedString - Convert the map passed in to a sorted key/value string
func MapToSortedString(mapData map[string]string) string {
	buffer := "["
	sortedKeys := make([]string, 0, len(mapData))
	for k := range mapData {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		buffer += k + "=" + mapData[k] + " "
	}
	buffer = strings.TrimSpace(buffer) + "]"
	return buffer
}

// RouteDataToMap - Extract from data map[] and return RouteData struct
func RouteDataToMap(routeData RouteData) map[string]string {
	dataMap := map[string]string{}
	dataMap[keyConnectUsingLB] = routeData.ConnectUsingLB
	dataMap[keyLoadBalancerIP] = routeData.LoadBalancerIP
	dataMap[keyLocalSubnet] = routeData.LocalSubnet
	dataMap[keyRemoteGateway] = routeData.RemoteGateway
	dataMap[keyRemoteSubnet] = routeData.RemoteSubnet
	dataMap[keyRouteTable] = routeData.RouteTable
	dataMap[keyVpnPodDevice] = routeData.VpnPodDevice
	dataMap[keyVpnPodIP] = routeData.VpnPodIP
	dataMap[keyVpnPodName] = routeData.VpnPodName
	dataMap[keyWorkerNodeIP] = routeData.WorkerNodeIP
	dataMap[keyWorkerSubnet] = routeData.WorkerSubnet
	return dataMap
}

// UpdateConfigMap - Update the config map with the specified routing data
func UpdateConfigMap(client *kubernetes.Clientset, namespace, configMapName string, routeData RouteData) {
	cm, err := getConfigMap(client, namespace, configMapName)
	if err != nil {
		log.Fatalf("ERROR: Failed to retrieve %s/%s: %v", namespace, configMapName, err)
	}
	cm.Data = RouteDataToMap(routeData)
	log.Printf("   config map data: %v", MapToSortedString(cm.Data))
	_, err = client.CoreV1().ConfigMaps(namespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
	if err != nil {
		log.Fatalf("ERROR: Failed to update/create config map: %v", err)
	}
}

// WatchConfigMap - watch for updates to config maps and calls the provided routines
func WatchConfigMap(client *kubernetes.Clientset, namespace, configMapName string,
	addFunc func(obj interface{}),
	deleteFunc func(obj interface{}),
	updateFunc func(oldObj, newObj interface{})) {
	// Create a watch on configMaps in kube-system
	log.Print("Create watchList for configMap changes")
	watchList := cache.NewListWatchFromClient(
		client.CoreV1().RESTClient(),
		corev1.ResourceConfigMaps.String(),
		namespace,
		fields.OneTermEqualSelector("metadata.name", configMapName))

	// Handler routine for add/delete/update events
	log.Print("Register handler routines for configMap changes")
	_, controller := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: watchList,
		ObjectType:    &corev1.ConfigMap{},
		ResyncPeriod:  0,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    addFunc,
			DeleteFunc: deleteFunc,
			UpdateFunc: updateFunc,
		},
	})

	stop := make(chan struct{})
	if controller != nil { // Should only be nil for unit tests
		go controller.Run(stop)
	}
}
