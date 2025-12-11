/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2023 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package monitoring

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
)

const (
	strChannel  = "channel"
	strUsername = "username"
	strIcon     = "icon_emoji"
	strText     = "text"
)

// sendSlackMessage - Post slack message
func sendSlackMessage(message string) int {
	if monitorCfg.SlackWebhook != "" {
		slackMessage := podLocationInfo + message
		jsonString := "{"
		if monitorCfg.SlackChannel != "" {
			jsonString += fmt.Sprintf("%q:%q,", strChannel, monitorCfg.SlackChannel)
		}
		if monitorCfg.SlackUsername != "" {
			jsonString += fmt.Sprintf("%q:%q,", strUsername, monitorCfg.SlackUsername)
		}
		if monitorCfg.SlackIcon != "" {
			jsonString += fmt.Sprintf("%q:%q,", strIcon, monitorCfg.SlackIcon)
		}
		jsonString += fmt.Sprintf("%q:%q}", strText, slackMessage)
		log.Printf("monitoring | Sending Slack message to URL %s", monitorCfg.SlackWebhook)
		res, err := http.Post(monitorCfg.SlackWebhook, "application/json", bytes.NewBuffer([]byte(jsonString)))
		if err != nil {
			log.Printf("monitoring | WARNING: Failed to send Slack message: %v", err)
			return -1
		}
		// Security fix for defer res.Body.Close()
		// G307 (CWE-703): Deferring unsafe method "Close" on type "io.ReadCloser" (Confidence: HIGH, Severity: MEDIUM)
		defer func() {
			if err := res.Body.Close(); err != nil {
				log.Fatalf("Error closing file: %v", err)
			}
		}()
		log.Printf("monitoring | HTTP status code: %d", res.StatusCode)
		return res.StatusCode
	}
	return 0
}
