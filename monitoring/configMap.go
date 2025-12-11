/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2022 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package monitoring

import (
	"log"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

const (
	defaultRetryTimes = 3
)

// readConfigFile - Read the configuration file into a struct
func readConfigFile(monitoringConfigFile string) {
	log.Printf("monitoring | Config file: %s", monitoringConfigFile)
	yamlFile, err := os.ReadFile(monitoringConfigFile) // #nosec G304 filename passed is always a fixed constant string
	if err != nil {
		log.Fatalf("monitoring | ERROR: Failed to open config file: %v", err)
	}
	err = yaml.Unmarshal(yamlFile, &monitorCfg)
	if err != nil {
		log.Fatalf("monitoring | ERROR: Un-marshalling config file: %v", err)
	}
	log.Printf("monitoring | Config: %+v", monitorCfg)
}

// Validate - used to do basic validation of monitoring parameter values and set them to default values if there is an issue.
func validate() {
	if monitorCfg.PrivateIPs != "" {
		verifyIP(strings.Split(monitorCfg.PrivateIPs, ","))
	}

	if monitorCfg.HTTPEndpoints != "" {
		verifyEndpoint(strings.Split(monitorCfg.HTTPEndpoints, ","))
	}

	if monitorCfg.PrivateIPs == "" && monitorCfg.HTTPEndpoints == "" {
		log.Fatalf("monitoring | ERROR: Either monitoring.privateIPs or monitoring.httpEndpoints must be configured")
	}

	if monitorCfg.Timeout <= 0 {
		log.Fatalf("monitoring | ERROR: monitoring.timeout is set to an incorrect value: %d", monitorCfg.Timeout)
	}

	if monitorCfg.Delay <= 0 {
		log.Fatalf("monitoring | ERROR: monitoring.delay is set to an incorrect value: %d", monitorCfg.Delay)
	}

	if monitorCfg.SlackWebhook != "" {
		rc := sendSlackMessage("Initializing monitor logic")
		if rc != 200 {
			log.Fatalf("monitoring | ERROR: Unable to post message to the Slack channel")
		}
	} else {
		log.Print("monitoring | Slack notifications have not been configured")
	}
}
