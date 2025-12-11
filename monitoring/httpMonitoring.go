/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2023 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package monitoring provides methods for monitoring the VPN connection
package monitoring

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// curlTest - call http get and repeat 'retryTest' times if there is an issue
func curlTest(itemToTest string, timeout int) (int, error) {
	status := -1
	var err error
	retryTest := defaultRetryTimes
	for retryTest >= 0 {
		result, status, errReturned := httpGet(timeout, itemToTest)
		if result {
			return status, nil
		}
		err = errReturned
		retryTest--
	}
	return status, err
}

// httpGET performs http GET requests
func httpGet(timeout int, url string) (bool, int, error) {
	httpTimeout := time.Duration(timeout) * time.Second
	client := http.Client{
		Timeout: httpTimeout,
	}
	if !strings.HasPrefix(url, "http") {
		url = strings.Replace(url, "", "http://", 1)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, -1, fmt.Errorf("%v", err)
	}
	res, err := client.Do(req)
	if res == nil {
		return false, -1, fmt.Errorf("%v", err)
	}
	// Security fix for defer res.Body.Close()
	// G307 (CWE-703): Deferring unsafe method "Close" on type "io.ReadCloser" (Confidence: HIGH, Severity: MEDIUM)
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Fatalf("Error closing file: %v", err)
		}
	}()
	return true, res.StatusCode, nil
}
