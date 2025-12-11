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
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var simpleKeysMustExist = []string{"authby", "auto", "left", "leftid", "leftsubnet", "right", "rightid", "rightsubnet"}
var strictKeysMustExist = []string{"forceencaps", "mobike"}
var simpleKeysIPAddr = []string{"left", "right"}
var simpleKeysSubnet = []string{"leftsubnet", "rightsubnet"}
var simpleKeysNumeric = []string{"keyingtries", "lifebytes", "lifepackets", "marginbytes", "marginpackets", "replay_window", "reqid"}
var simpleKeysDuration = []string{"dpddelay", "dpdtimeout", "inactivity", "ikelifetime", "keylife", "lifetime", "margintime", "rekeymargin", "crlcheckinterval", "keep_alive"}
var sedCommand = string("/bin/sed")

type keyValidSet struct {
	key string
	set []string
}

var simpleKeyValidSets = []keyValidSet{

	// ipsec.conf: config setup
	{"cachecrls", []string{"yes", "no"}},
	{"charonstart", []string{"yes", "no"}},
	{"strictcrlpolicy", []string{"yes", "ifuri", "no"}},
	{"uniqueids", []string{"yes", "no", "never", "replace", "keep"}},

	// Old options (before 5.0.0)
	{"nat_traversal", []string{"yes", "no"}},
	{"nocrsend", []string{"yes", "no"}},
	{"pkcs11keepstate", []string{"yes", "no"}},
	{"pkcs11proxy", []string{"yes", "no"}},
	{"plutostart", []string{"yes", "no"}},

	// General Connection Parameters
	{"aggressive", []string{"yes", "no"}},
	{"authby", []string{"pubkey", "rsasig", "ecdsasig", "psk", "secret", "xauthrsasig", "xauthpsk", "never"}},
	{"auto", []string{"ignore", "add", "route", "start"}},
	{"closeaction", []string{"none", "clear", "hold", "restart"}},
	{"compress", []string{"yes", "no"}},
	{"dpdaction", []string{"none", "clear", "hold", "restart"}},
	{"forceencaps", []string{"yes", "no"}},
	{"fragmentation", []string{"yes", "accept", "force", "no"}},
	{"installpolicy", []string{"yes", "no"}},
	{"keyexchange", []string{"ike", "ikev1", "ikev2"}},
	{"mobike", []string{"yes", "no"}},
	{"modeconfig", []string{"push", "pull"}},
	{"reauth", []string{"yes", "no"}},
	{"rekey", []string{"yes", "no"}},
	{"sha256_96", []string{"yes", "no"}},
	{"type", []string{"tunnel", "transport", "transport_proxy", "passthrough", "drop"}},
	{"xauth", []string{"client", "server"}},

	// left | right End Parameters
	{"leftallowany", []string{"yes", "no"}},
	{"leftauth", []string{"pubkey", "psk", "eap", "xauth"}},
	{"leftfirewall", []string{"yes", "no"}},
	{"leftsendcert", []string{"never", "no ", "ifasked", "always", "yes"}},
	{"rightallowany", []string{"yes", "no"}},
	{"rightauth", []string{"pubkey", "psk", "eap", "xauth"}},
	{"rightfirewall", []string{"yes", "no"}},
	{"rightsendcert", []string{"never", "no ", "ifasked", "always", "yes"}},

	// IKEv2 Mediation Extension Parameters
	{"mediation", []string{"yes", "no"}},

	// Removed parameters (since 5.0.0)
	{"auth", []string{"esp", "ah"}},
	{"pfs", []string{"yes", "no"}},
}

var strictKeyValidSets = []keyValidSet{
	{"authby", []string{"psk", "secret"}},
	{"auto", []string{"add", "start"}},
	{"forceencaps", []string{"yes"}},
	{"mobike", []string{"no"}},
	{"type", []string{"tunnel"}},
	{"left", []string{"%any"}},
}

// Environment variable constants
const (
	envVarValidateConfig  = "VALIDATE_CONFIG"
	envVarPrivateIPToPing = "PRIVATE_IP_TO_PING"
	envVarRemoteSubnetNAT = "REMOTE_SUBNET_NAT"
)

// ConfigData - Map used to hold configuration key/values from the config file
type ConfigData map[string][]string

// List of ConfigData keys used by the main strongswan logic to extract specific fields
const (
	ConfigDataLeftID        string = "leftid"          // Value found for leftid
	ConfigDataLeftSubnet    string = "leftsubnet"      // Value found for leftsubnet
	ConfigDataRightSubnet   string = "rightsubnet"     // Value found for rightsubnet
	ConfigDataRemoteGateway string = "right"           // Value found for right
	ConfigDataIpsecAuto     string = "auto"            // Value found for auto
	LeftIDNodePublicIP      string = "%nodePublicIP"   // Constant for leftid
	LeftIDLoadBalancerIP    string = "%loadBalancerIP" // Constant for leftid
	LeftSubnetZoneSpecific  string = "%zoneSubnet"     // Constant for leftsubnet
)

// validateType - Type of validation that should be done on key's value
type validateType string

// List of validation types - used when calling validateValueIsCorrect()
const (
	valueDuration validateType = "duration" // Duration value
	valueIPAddr   validateType = "ip_addr"  // IP address value
	valueNumeric  validateType = "numeric"  // Numeric value
	valueSubnet   validateType = "subnet"   // IP subnet value
)

// CopyConfigFile - Copy config files over to the correct location
func CopyConfigFile(filename, sourceDir, targetDir string, required bool) {
	source := filepath.Join(sourceDir, filename)
	target := filepath.Join(targetDir, filename)
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		log.Printf("Copying %s ...", source)
		if runtime.GOOS == "darwin" {
			err = exec.Command("cp", source, target).Run() // #nosec G204 variables used are hard coded compile time constants
		} else {
			err = exec.Command("sudo", "/bin/cp", source, target).Run() // #nosec G204 variables used are hard coded compile time constants
		}
		if err != nil {
			log.Fatalf("ERROR: Failed to copy %s to %s: %v", source, target, err)
		}
	} else if required {
		log.Fatalf("ERROR: Required configuration file: %s was not found", source)
	}
}

// ExtractConfigData - Build a "," separated list of values for the specified config map key
func ExtractConfigData(configData ConfigData, key string) string {
	retVal := ""
	for _, value := range configData[key] {
		if len(retVal) > 0 {
			retVal += ","
		}
		retVal += value
	}
	return retVal
}

// Helper function to check if key is in the config map
func isKeyInConfig(configData ConfigData, key string) bool {
	_, exist := configData[key]
	return exist
}

// readConfigFile - Read the configuration file
func readConfigFile(filename string) ConfigData {
	configData := ConfigData{}

	file, err := os.Open(filename) // #nosec G304 filename passed is always a fixed constant string
	if err != nil {
		log.Fatalf("ERROR: Failed to open %s: %v", filename, err)
	}
	defer file.Close() // #nosec G307 safe to call close() here

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("%s", line)

		// Strip all comments from the config file
		if comment := strings.Index(line, "#"); comment >= 0 {
			line = line[:comment]
		}
		// Strip beginning and trailing spaces
		line = strings.TrimSpace(line)

		// If the line has does not have "=", skip ir
		equal := strings.Index(line, "=")
		if equal <= 0 {
			continue
		}

		// Verify no spaces are to the left of the "="
		space := strings.Index(line, " ")
		if space > 0 && space < equal {
			continue
		}

		// Determine key (line to the left of =)
		key := line[:equal]
		value := ""

		// Make sure we don't run off end of line
		if len(line) > equal+1 {
			value = line[equal+1:]
		}

		// Update the config map
		if _, exist := configData[key]; exist {
			configData[key] = append(configData[key], value)
		} else {
			mapData := []string{value}
			configData[key] = mapData
		}
	}
	return configData
}

// UpdateConfigLeftID - Update the leftid property in the config file if needed
func UpdateConfigLeftID(filename, leftID, publicIP, loadBalancerIP string) {
	newLeftID := ""
	switch leftID {
	case LeftIDNodePublicIP:
		newLeftID = publicIP
	case LeftIDLoadBalancerIP:
		newLeftID = loadBalancerIP
	default:
		return
	}
	// Make sure the new leftid value is an IP address
	if net.ParseIP(newLeftID) == nil {
		log.Fatalf("ERROR: Generated local.id value is not an IP address: %s", newLeftID)
	}
	// Update the ipsec.conf file
	updateConfigFile(filename, "leftid", leftID, newLeftID)
}

// UpdateConfigLeftSubnet - Update the leftsubnet property in the config file if needed
func UpdateConfigLeftSubnet(filename, leftSubnet, zoneSubnet string) {
	// Only update the config file if leftsubnet is set to special value
	if leftSubnet != LeftSubnetZoneSpecific {
		return
	}
	// Update the ipsec.conf file
	updateConfigFile(filename, "leftsubnet", leftSubnet, zoneSubnet)
}

// updateConfigFile - Update the keyword setting in filename from existing to newValue
func updateConfigFile(filename, keyword, existing, newValue string) {
	// Update the ipsec.conf file
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Fatalf("ERROR: %v", err)
	}
	log.Printf("Updating [%s] from [%s] to [%s]", keyword, existing, newValue)
	// A "-" is used for the sed delimiter instead of "/" because "newValue" will contain "/" for keyword=leftsubnet
	pattern := fmt.Sprintf("s-%s=%s-%s=%s-g", keyword, existing, keyword, newValue)
	log.Printf("sedCommand: %s\n", sedCommand)
	var outBytes []byte
	var err error
	if runtime.GOOS == "darwin" {
		outBytes, err = exec.Command("sed", "-i", "-e", pattern, filename).CombinedOutput() // #nosec G204 variables used are hard coded compile time constants
	} else {
		outBytes, err = exec.Command("sudo", sedCommand, "-i", "-e", pattern, filename).CombinedOutput() // #nosec G204 variables used are hard coded compile time constants
	}
	if err != nil {
		log.Fatalf("ERROR: sed failed: %s - %v", string(outBytes), err)
	}
}

// verifyPrivateIP - Verify the environment variable PRIVATE_IP_TO_PING if it was specified
func verifyPrivateIP(configData ConfigData) []error {
	errorList := make([]error, 0)
	privateIP := os.Getenv(envVarPrivateIPToPing)
	remoteSubnetNAT := os.Getenv(envVarRemoteSubnetNAT)
	if privateIP != "" {
		ip := net.ParseIP(privateIP)
		if ip == nil {
			errorMsg := fmt.Errorf("invalid IP address: %s=%s ", envVarPrivateIPToPing, privateIP)
			errorList = append(errorList, errorMsg)
		} else {
			found := false
			for _, subnet := range strings.Split(ExtractConfigData(configData, "rightsubnet"), ",") {
				_, ipnet, err := net.ParseCIDR(subnet)
				if err == nil && ipnet.Contains(ip) {
					found = true
				}
			}
			// If we are doing remoteSubnetNATs, check if the IP is in there
			if remoteSubnetNAT != "" {
				for _, rule := range strings.Split(remoteSubnetNAT, ",") {
					mapped := strings.Split(rule, "=")[1]
					_, ipnet, err := net.ParseCIDR(mapped)
					if err == nil && ipnet.Contains(ip) {
						found = true
					}
				}
			}

			if !found {
				errorMsg := fmt.Errorf("IP address %s not in remote subnet %v", privateIP, configData["rightsubnet"])
				errorList = append(errorList, errorMsg)
			}
		}
	}
	return errorList
}

// verifyValidateConfig - Verify the setting of the environment variable VALIDATE_CONFIG
func verifyValidateConfig() string {
	log.Print("Retrieve config validation setting...")
	validateConfig := os.Getenv(envVarValidateConfig)
	log.Printf("    %s=%s", envVarValidateConfig, validateConfig)
	switch validateConfig {
	case "simple":
		log.Print("Simple config validation will be done.")
	case "strict":
		log.Print("Strict config validation will be done.")
	case "off":
		log.Print("No config validation will be done.")
	case "":
		validateConfig = "strict"
		log.Printf("%s was not defined.  Defaulting to: %s config validation", envVarValidateConfig, validateConfig)
	default:
		log.Fatalf("ERROR: Invalid environment variable: %s=%s  Valid choices: [ off, simple, strict ]", envVarValidateConfig, validateConfig)
	}
	return validateConfig
}

// ValidateConfig - Validate the specified configuration file and return the list of remote subnets
func ValidateConfig(filename string) ConfigData {
	log.Printf("Read the configuration settings in %s ...", filename)
	configData := readConfigFile(filename)
	validateConfig := verifyValidateConfig()
	if validateConfig == "off" {
		return configData
	}
	simpleErrors := verifyPrivateIP(configData)
	strictErrors := make([]error, 0)

	// Simple validation steps
	for _, key := range simpleKeysMustExist {
		error := validateKeyExists(configData, key)
		if error != nil {
			simpleErrors = append(simpleErrors, error)
		}
	}
	for _, key := range simpleKeysDuration {
		errors := validateValueIsCorrect(configData, key, valueDuration)
		if errors != nil {
			simpleErrors = append(simpleErrors, errors...)
		}
	}
	for _, key := range simpleKeysIPAddr {
		errors := validateValueIsCorrect(configData, key, valueIPAddr)
		if errors != nil {
			simpleErrors = append(simpleErrors, errors...)
		}
	}
	for _, key := range simpleKeysNumeric {
		errors := validateValueIsCorrect(configData, key, valueNumeric)
		if errors != nil {
			simpleErrors = append(simpleErrors, errors...)
		}
	}
	for _, key := range simpleKeysSubnet {
		errors := validateValueIsCorrect(configData, key, valueSubnet)
		if errors != nil {
			simpleErrors = append(simpleErrors, errors...)
		}
	}
	for _, valid := range simpleKeyValidSets {
		errors := validateValueFromSet(configData, valid.key, valid.set)
		if errors != nil {
			simpleErrors = append(simpleErrors, errors...)
		}
	}

	// Strict validation steps
	if validateConfig == "strict" {
		for _, key := range strictKeysMustExist {
			error := validateKeyExists(configData, key)
			if error != nil {
				strictErrors = append(strictErrors, error)
			}
		}
		for _, valid := range strictKeyValidSets {
			errors := validateValueFromSet(configData, valid.key, valid.set)
			if errors != nil {
				strictErrors = append(strictErrors, errors...)
			}
		}
		// Additional IKEv1 validation checks
		if ExtractConfigData(configData, "keyexchange") == "ikev1" {
			if !isKeyInConfig(configData, "esp") {
				errorMsg := fmt.Errorf("ipsec.esp must be specified if ipsec.keyexchange=ikev1")
				strictErrors = append(strictErrors, errorMsg)
			}
			if !isKeyInConfig(configData, "ike") {
				errorMsg := fmt.Errorf("ipsec.ike must be specified if ipsec.keyexchange=ikev1")
				strictErrors = append(strictErrors, errorMsg)
			}
			if strings.Contains(ExtractConfigData(configData, "leftsubnet"), ",") {
				errorMsg := fmt.Errorf("local.subnet must only contain a single subnet if ipsec.keyexchange=ikev1")
				strictErrors = append(strictErrors, errorMsg)
			}
			if strings.Contains(ExtractConfigData(configData, "rightsubnet"), ",") {
				errorMsg := fmt.Errorf("remote.subnet must only contain a single subnet if ipsec.keyexchange=ikev1")
				strictErrors = append(strictErrors, errorMsg)
			}
		}
	}

	// Display results of the validation tests
	log.Printf("Simple validation errors: %d", len(simpleErrors))
	for _, error := range simpleErrors {
		log.Printf("   - %s", error.Error())
	}
	log.Printf("Strict validation errors: %d", len(strictErrors))
	for _, error := range strictErrors {
		log.Printf("   - %s", error.Error())
	}

	// If validation errors were detected, exit
	if len(simpleErrors) > 0 || len(strictErrors) > 0 {
		log.Fatalf("ERROR: Total errors detected: %d", len(simpleErrors)+len(strictErrors))
	}
	return configData
}

// validateKeyExists - Validate that the specified key was found in the configuration map
func validateKeyExists(configData ConfigData, key string) error {
	if !isKeyInConfig(configData, key) {
		return fmt.Errorf("required setting not found: %s= ", key)
	}
	return nil
}

// validateValueFromSet - Validate that the values for the specified key are valid
func validateValueFromSet(configData ConfigData, key string, validSet []string) []error {
	// If the specified key in not in the config data, then just return
	if !isKeyInConfig(configData, key) {
		return nil
	}

	// Create valid map from the valid set (makes following logic easier)
	validMap := map[string]bool{}
	for _, valid := range validSet {
		validMap[valid] = true
	}

	errorsDetected := make([]error, 0)
	// For each occurrence of the key
	for _, value := range configData[key] {

		// Check to see if value is valid
		if !validMap[value] {
			errorMsg := fmt.Errorf("invalid key value: %s=%s  Valid choices: %v", key, value, validSet)
			errorsDetected = append(errorsDetected, errorMsg)
		}
	}
	if len(errorsDetected) > 0 {
		return errorsDetected
	}
	return nil
}

// validateValueIsCorrect - Validate that the key value is correct
func validateValueIsCorrect(configData ConfigData, key string, valueType validateType) []error {
	// If the specified key in not in the config data, then just return
	if !isKeyInConfig(configData, key) {
		return nil
	}

	errorsDetected := make([]error, 0)
	// For each occurrence of the key
	for _, value := range configData[key] {
		switch valueType {
		case valueDuration:
			length := len(value)
			if length > 0 {
				orgValue := value
				lastChar := value[length-1]
				if lastChar == byte('s') || lastChar == byte('m') || lastChar == byte('h') {
					value = value[:length-1]
				}
				_, error := strconv.Atoi(value)
				if error != nil {
					errorMsg := fmt.Errorf("invalid duration value: %s=%s ", key, orgValue)
					errorsDetected = append(errorsDetected, errorMsg)
				}
			}
		case valueNumeric:
			if key == "keyingtries" && value == "%forever" {
				continue
			}
			_, error := strconv.Atoi(value)
			if error != nil {
				errorMsg := fmt.Errorf("invalid numeric value: %s=%s ", key, value)
				errorsDetected = append(errorsDetected, errorMsg)
			}
		case valueIPAddr:
			if value == "%any" || value == "%defaultroute" {
				continue
			}
			if strings.ContainsAny(value, "-/%,") || net.ParseIP(value) == nil {
				errorMsg := fmt.Errorf("invalid IP address: %s=%s ", key, value)
				errorsDetected = append(errorsDetected, errorMsg)
			}
		case valueSubnet:
			for _, subnet := range strings.Split(value, ",") {
				if subnet == LeftSubnetZoneSpecific {
					continue
				}
				_, _, error := net.ParseCIDR(subnet)
				if error != nil || strings.ContainsAny(subnet, "%[]") {
					errorMsg := fmt.Errorf("invalid IP subnet: %s=%s ", key, value)
					errorsDetected = append(errorsDetected, errorMsg)
				}
			}
		}
	}
	if len(errorsDetected) > 0 {
		return errorsDetected
	}
	return nil
}
