/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package monitoring

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// pingTest - call ping
func pingTest(itemToTest string, timeout int) error {
	output, err := exec.Command("ping", itemToTest, fmt.Sprintf("-c%d", defaultRetryTimes), fmt.Sprintf("-w%d", timeout)).CombinedOutput() // #nosec G204 variables are built from fixed constants and network information, user can not override
	if err != nil {
		// Typical failure results from ping:
		//
		// PING 10.73.46.119 (10.73.46.119): 56 data bytes
		// --- 10.73.46.119 ping statistics ---
		// 5 packets transmitted, 0 packets received, 100% packet loss
		//
		// Return the last line that has length > 0 from the ping output
		lastLine := ""
		for _, line := range strings.Split(string(output), "\n") {
			if len(strings.TrimSpace(line)) > 0 {
				lastLine = strings.TrimSpace(line)
			}
		}
		return errors.New(lastLine)
	}
	return nil
}
