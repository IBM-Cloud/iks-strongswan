/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
 * The source code for this program is not published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package utils provides common GO helper routines
package utils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
)

// ShellCommand - parameters need to control/track execution of system call
type ShellCommand struct {
	Path          string
	Name          string
	Args1         []string // Arguments to pass to the system call at start time (before the callerArgs)
	Args2         []string // Arguments to pass to the system call at start time (after the callerArgs)
	MonitorOutput bool     // Display stdout/stderr?
	RunAsRoot     bool

	running   bool // Is the system call still running?
	cmd       *exec.Cmd
	cmdReader io.ReadCloser
}

// Start the command
func (sh *ShellCommand) Start(callerArgs ...string) {
	args := append(append(sh.Args1, callerArgs...), sh.Args2...) // Combine all the args
	run := sh.Path + sh.Name
	if sh.RunAsRoot && runtime.GOOS != "darwin" {
		args = append([]string{run}, args...)  // sudo is now the command and the command becomes an arg
		sh.cmd = exec.Command("sudo", args...) // #nosec G204 variables used are hard coded compile time constants
		log.Printf("Running command with sudo: %+v\n", sh.cmd)
	} else {
		sh.cmd = exec.Command(run, args...) // #nosec G204 variables used are hard coded compile time constants
	}
	if sh.MonitorOutput {
		var err error
		sh.cmdReader, err = sh.cmd.StdoutPipe()
		if err != nil {
			log.Fatalf("ERROR: Creating StdoutPipe: %v", err)
		}
		sh.cmd.Stderr = sh.cmd.Stdout
	} else {
		sh.cmd.Stdout = os.Stdout
		sh.cmd.Stderr = os.Stderr
	}

	log.Printf("Starting %s ...", sh.Path+sh.Name)
	rErr := sh.cmd.Start()
	if rErr != nil {
		log.Fatalf("ERROR: Failed to start %s: %v", sh.Name, rErr)
	}
	sh.running = true
}

// Monitor the output of the command
func (sh *ShellCommand) Monitor(filterOutput func(string) (bool, string), parseOutput func(string)) {
	if sh.MonitorOutput && sh.running {
		scanner := bufio.NewScanner(sh.cmdReader)
		for scanner.Scan() {
			line := scanner.Text()
			printLine := true
			// If a filter routine was provided, call it
			if filterOutput != nil {
				printLine, line = filterOutput(line)
			}
			// If the filter routine returned "false", don't display line
			if printLine {
				log.Printf("%s | %s", sh.Name, line)
			}
			// If a parse routine was provided, call it
			if parseOutput != nil {
				parseOutput(line)
			}
		}
	}
}

// Stop the command
func (sh *ShellCommand) Stop() {
	if sh.running {
		log.Printf("Stopping %s ...", sh.Path+sh.Name)
		pid := fmt.Sprintf("%v", sh.cmd.Process.Pid)
		var err error
		if runtime.GOOS == "darwin" {
			_, err = exec.Command("kill", "-15", pid).CombinedOutput() // #nosec G204 variable must be used because pid is unknown at compile time
		} else {
			_, err = exec.Command("sudo", "/bin/kill", "-15", pid).CombinedOutput() // #nosec G204 variable must be used because pid is unknown at compile time
		}
		if err != nil {
			log.Fatalf("ERROR: Failed to send SIGTERM to %s: %v", sh.Name, err)
		}
		sh.running = false
	}
}

// Wait for the command to end
func (sh *ShellCommand) Wait() {
	log.Printf("Waiting for %s to end...", sh.Path+sh.Name)
	err := sh.cmd.Wait()
	if err != nil {
		log.Printf("WARNING: Failed while waiting for %s to end: %v", sh.Name, err)
	}
	log.Printf("%s has ended", sh.Path+sh.Name)
}
