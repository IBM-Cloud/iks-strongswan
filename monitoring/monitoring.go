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
	"fmt"
	"log"
	"strings"
	"time"
)

// monitorYamlConfig - struct that contains the monitoring configuration information
type monitorYamlConfig struct {
	ClusterName   string `yaml:"clusterName"`
	Delay         int    `yaml:"delay"`
	HTTPEndpoints string `yaml:"httpEndpoints"`
	PrivateIPs    string `yaml:"privateIPs"`
	SlackWebhook  string `yaml:"slackWebhook"`
	SlackChannel  string `yaml:"slackChannel"`
	SlackUsername string `yaml:"slackUsername"`
	SlackIcon     string `yaml:"slackIcon"`
	Timeout       int    `yaml:"timeout"`
}

type monitorTestResults struct {
	curl string
	ping string
}

const (
	monConfigFile = "/etc/ipsec.config/monitoring.conf"
)

var (
	monitorActive         bool
	monitorCfg            monitorYamlConfig
	monitorResult         monitorTestResults
	podLocationInfo       string
	stopMonitor           bool
	stopMonitorThreadChan = make(chan string, 1)
	timeToLive            int64
	timeToRunMonitoring   int64
)

// Start - request to start the monitoring testing
func Start() {
	timeToRunMonitoring = time.Now().Unix() + 2 // Add 2 seconds to allow VPN tunnel to be ready
	stopMonitor = false
	monitorActive = true
}

// Stop - request to stop the monitoring testing
func Stop() {
	timeToLive = time.Now().Unix() + 10 // Add 10 second delay before indicating that the VPN is down
	stopMonitor = true
}

// Init - used to setup the monitoring struct and validate the struct, and start the monitoring goroutine
func Init(podName, clusterID string) {
	readConfigFile(monConfigFile)
	podLocationInfo = podLocation(podName, monitorCfg.ClusterName, clusterID)
	validate()
	go monitorThread()
}

// Cancel - used to stop the monitoring. Called when the VPN pod is ending
func Cancel() {
	stopMonitorThreadChan <- "Stop"
	message := "VPN pod is exiting.  VPN is going down"
	log.Printf("monitoring | %s", message)
	sendSlackMessage(message)
}

// monitorThread - code to run the monitoring
func monitorThread() {
	for {
		if monitorActive {
			if time.Now().Unix() >= timeToRunMonitoring {
				execute()
				timeToRunMonitoring = time.Now().Unix() + int64(monitorCfg.Delay)
			}
			if stopMonitor && (time.Now().Unix() >= timeToLive) {
				monitorResult.curl = ""
				monitorResult.ping = ""
				monitorActive = false
				stopMonitor = false
				message := "VPN is down"
				log.Printf("monitoring | %s", message)
				sendSlackMessage(message)
			}
		}
		select {
		case <-stopMonitorThreadChan:
			log.Print("monitoring | VPN monitor thread is exiting")
			return
		case <-time.After(time.Duration(2) * time.Second):
		}
	}
}

// execute - executes the monitoring tests
func execute() {
	log.Print("monitoring | Test network connectivity over the VPN")
	message := ""
	if monitorCfg.PrivateIPs != "" {
		ipOutput := runTest("ping", monitorCfg.PrivateIPs)
		if monitorResult.ping != ipOutput {
			monitorResult.ping = ipOutput
			message += ipOutput
		}
	}
	if monitorCfg.HTTPEndpoints != "" {
		curlOutput := runTest("curl", monitorCfg.HTTPEndpoints)
		if monitorResult.curl != curlOutput {
			monitorResult.curl = curlOutput
			message += curlOutput
		}
	}

	// If ping or curl output has changed, then message will be set
	if message != "" {
		if strings.Contains(message, "Success") {
			message += "VPN is up"
		} else {
			message += "Monitoring tests failed. VPN may be down"
		}
		sendSlackMessage(message)
	}
}

// runTest - runs the monitoring tests and returns the results
func runTest(testType, varToParse string) string {
	var err error
	resultMessage := ""
	itemsToTest := strings.Split(varToParse, ",")
	for _, itemToTest := range itemsToTest {
		result := ""
		switch testType {
		case "ping":
			result += fmt.Sprintf("Pinging remote %s. ", itemToTest)
			err = pingTest(itemToTest, monitorCfg.Timeout)
		case "curl":
			result += fmt.Sprintf("Curling remote %s. ", itemToTest)
			var status int
			status, err = curlTest(itemToTest, monitorCfg.Timeout)
			result += fmt.Sprintf("Response status code [%d]. ", status)
		}
		if err != nil {
			result += fmt.Sprintf("Failed with: %v\n", err)
		} else {
			result += "Success\n"
		}
		log.Printf("monitoring | %s", result)
		resultMessage += result
	}
	return resultMessage
}

// podLocation - generates the pod location information that is added to the slack notifications
func podLocation(podName, clusterName, clusterID string) string {
	podInfo := fmt.Sprintf("VPN pod %s", podName)
	if clusterName != "" {
		podInfo += fmt.Sprintf(" in cluster %s", clusterName)
		if clusterID != "" {
			podInfo += fmt.Sprintf(" (%s)", clusterID)
		}
	} else if clusterID != "" {
		podInfo += fmt.Sprintf(" in %s", clusterID)
	}
	return podInfo + "\n"
}
