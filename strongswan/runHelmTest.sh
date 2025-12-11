#!/bin/bash
#*******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021, 2024 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Perform the correct helm test operation
case $1 in
    "check-state")
        echo "-------------------------------------------------------"
        echo "  Check the state of the ipsec service in the VPN pod"
        echo "-------------------------------------------------------"
        cd /tmp || exit
        echo "-- Add kubectl to the image -- ${KUBE_VERSION}"
        curl -LO "https://dl.k8s.io/release/${KUBE_VERSION}/bin/linux/amd64/kubectl"
        chmod +x kubectl
        echo "-- Determine vpn pod name --"
        vpnPod=$(/tmp/kubectl get pod -n "${NAMESPACE}" -l "app=strongswan,release=${RELEASE_NAME}" --no-headers | awk '{ print $1 }')
        echo "VPN pod: $vpnPod"
        echo "-- Retrieve status of the VPN pod --"
        /tmp/kubectl exec -n "${NAMESPACE}" "${vpnPod}" -- sudo /usr/sbin/ipsec statusall | tee /tmp/ipsec.status
        grep -q "ESTABLISHED" /tmp/ipsec.status
        ;;

    "ping-remote-gw")
        echo "----------------------------------------------"
        echo "  Verify that we can PING the remote gateway"
        echo "----------------------------------------------"
        echo "Configuration values:"
        echo "   ipsec.auto = ${IPSEC_AUTO}"
        echo "   remote.gateway = ${REMOTE_GATEWAY}"
        if [ "${REMOTE_GATEWAY}" == "%any" ]; then
            echo "Remote gateway was not configured."
            exit 1
        fi
        ping -c 3 "${REMOTE_GATEWAY}"
        ;;

    "ping-remote-ip-1")
        echo "-------------------------------------------------"
        echo "  Verify that we can ping the remote private ip"
        echo "-------------------------------------------------"
        echo "Configuration values:"
        echo "   local.subnet = ${LOCAL_SUBNET}"
        echo "   remote.subnet = ${REMOTE_SUBNET}"
        echo "   remote.privateIPtoPing = ${REMOTE_PRIVATE_IP_TO_PING}"
        echo "Local IP address: "
        ip -f inet -o addr | grep -v " lo " | awk '{ print "   "$2" = "$4 }'
        echo "-------------------------------------------------"
        if [ "${REMOTE_PRIVATE_IP_TO_PING}" == "" ]; then
            echo "Remote private IP was not configured"
            exit 1
        fi
        if ! echo "${LOCAL_SUBNET}" | grep -q "172.30.0.0/16"; then
            echo "In order for this ping test to work, the Kubernetes pod subnet (172.30.0.0/16)"
            echo "must be listed in local.subnet= that are exposed over the VPN  --AND--"
            echo "the on-prem VPN needs to accept all of the subnets listed in local.subnet="
            echo "-------------------------------------------------"
        fi
        ping -c 3 "${REMOTE_PRIVATE_IP_TO_PING}"
        ;;

    "ping-remote-ip-2")
        echo "-------------------------------------------------"
        echo "  Verify that we can ping the remote private ip"
        echo "-------------------------------------------------"
        echo "Configuration values:"
        echo "   local.subnet = ${LOCAL_SUBNET}"
        echo "   remote.subnet = ${REMOTE_SUBNET}"
        echo "   remote.privateIPtoPing = ${REMOTE_PRIVATE_IP_TO_PING}"
        echo "Local IP addresses: "
        ip -f inet -o addr | grep -v docker | grep -v tunl0 | grep -v " lo " | grep -v "/32" | awk '{ print "   "$2" = "$4 }'
        echo "-------------------------------------------------"
        if [ "${REMOTE_PRIVATE_IP_TO_PING}" == "" ]; then
            echo "Remote private IP was not configured"
            exit 1
        fi
        if ! echo "${LOCAL_SUBNET}" | grep -q "10."; then
            echo "In order for this ping test to work, the worker nodes private subnet (10.x.x.x)"
            echo "must be listed in local.subnet= that are exposed over the VPN  --AND--"
            echo "the on-prem VPN needs to accept all of the subnets listed in local.subnet="
            echo "-------------------------------------------------"
        fi
        ping -c 3 "${REMOTE_PRIVATE_IP_TO_PING}"
        ;;

    *)
        echo "Invalid argument specified"
        exit 1
        ;;
esac
