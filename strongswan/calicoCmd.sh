#!/bin/bash
#*******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * Copyright IBM Corp. 2018, 2023 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Calico IPPool
calico_IPPool="
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: %s
spec:
  cidr: %s
  disabled: true
  ipipMode: CrossSubnet
  nat-outgoing: false
"

if [ -f /tmp/calicoEnv.sh ]; then
    # shellcheck disable=SC1091
    source /tmp/calicoEnv.sh
fi

tmpFile=/tmp/calicoCmd.tmp

function check_set_ip_forward() {
    result=$(sudo /sbin/sysctl -a 2>&1 | grep ".ip_forward " | awk '{ print $3 }')
    if [ "$result" == "1" ]; then
        return
    fi
    sudo /sbin/sysctl -w net.ipv4.ip_forward=1 >/dev/null
    result=$(sudo /sbin/sysctl -a 2>&1 | grep ".ip_forward " | awk '{ print $3 }')
    if [ "$result" == "0" ]; then
        echo "Set 'privilegedVpnPod' to 'true' in the helm configuration"
        exit
    fi
}

function run_calicoctl() {
    # shellcheck disable=SC2086
    if ! calicoctl --allow-version-mismatch $1 >$tmpFile 2>&1; then
        cat $tmpFile
        exit 1
    fi
}

if [ "$1" == "createIPPool" ] && [ "$2" != "" ]; then
    calicoctl get ippool -o wide --allow-version-mismatch >/tmp/ippool.list
    if grep -q " $2 " /tmp/ippool.list; then
        ipPool=$(grep " $2 " /tmp/ippool.list | awk '{ print $1 }')
        calicoctl delete ippool "$ipPool" --allow-version-mismatch
    fi
    ipAddr=$(echo "$2" | tr './' '-')
    suffix=$(echo "$HOSTNAME" | rev | cut -d'-' -f1 | rev)
    # shellcheck disable=SC2059
    printf "$calico_IPPool" "$ipAddr-$suffix" "$2" >/tmp/calico-ippool.yaml
    calicoctl apply -f /tmp/calico-ippool.yaml --allow-version-mismatch

elif [ "$1" == "deleteIPPool" ] && [ "$2" != "" ]; then
    ipAddr=$(echo "$2" | tr './' '-')
    suffix=$(echo "$HOSTNAME" | rev | cut -d'-' -f1 | rev)
    calicoctl delete ippool "$ipAddr-$suffix" --allow-version-mismatch
    exit 0

elif [ "$1" == "getIPPool" ]; then
    calicoctl get ippool -o wide --allow-version-mismatch

elif [ "$1" == "getNodeSubnet" ] && [ "$2" != "" ]; then
    run_calicoctl "get node -o wide"
    grep "$2/" $tmpFile | awk '{ print $3 }'

elif [ "$1" == "getPodInterface" ]; then
    if ! echo "$HOSTNAME" | grep -q "strongswan"; then
        echo "The $1 option is only valid if we are running in the VPN strongswan pod"
        exit 1
    fi
    check_set_ip_forward
    run_calicoctl "get workloadendpoint --namespace=$NAMESPACE"
    grep "$HOSTNAME" $tmpFile | awk '{ print $5 }'

elif [ -z "$1" ] || [ "$1" == "-h" ]; then
    echo "calicoCmd  [option]"
    echo
    echo "Options:"
    echo "   createIPPool     subnet    Create ippool resource for the specified subnet"
    echo "   deleteIPPool     subnet    Delete ippool resource for the specified subnet"
    echo "   getIPPool                  Display list of ippool resources"
    echo "   getNodeSubnet    workerIP  Retrieve the private IP / subnet length of a specific worker node"
    echo "   getPodInterface            Retrieve the cali interface for the given pod (must be in VPN pod)"
    echo
    exit 0
else
    echo "Valid option was not specified.   Specify: -h for help"
    exit 1
fi
