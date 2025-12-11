#!/bin/bash
#*******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

install_kubectl() {
    if [ -f /tmp/kubectl ]; then
        return
    fi
    echo "Installing kubectl ..."
    cd /tmp || exit
    echo
    curl -LO https://dl.k8s.io/release/"$KUBE_VERSION"/bin/linux/amd64/kubectl
    chmod +x kubectl
    echo
    sleep 1
}

autoStart=$(grep "auto=" /etc/ipsec.conf | awk '{ print $1 }' | grep "^auto=" | cut -d'=' -f2)
localSubnet=$(grep "leftsubnet=" /etc/ipsec.conf | awk '{ print $1 }' | grep "^leftsubnet=" | cut -d'=' -f2 | tr '\n' ',')
remoteGateway=$(grep "right=" /etc/ipsec.conf | awk '{ print $1 }' | grep "^right=" | cut -d'=' -f2)
remoteSubnet=$(grep "rightsubnet=" /etc/ipsec.conf | awk '{ print $1 }' | grep "^rightsubnet=" | cut -d'=' -f2 | tr '\n' ',')

remoteIpSecActive=false

echo "----------------------------------------------------------------------"
echo "vpnDebug: Tool to help debug VPN connectivity issues"
echo
echo "Output lines that begin with: ERROR, WARNING, VERIFY, or CHECK"
echo "indicate possible errors with the VPN connectivity."
echo "----------------------------------------------------------------------"

fatalError=false
if [ "$RELEASE_NAME" == "" ]; then
    echo "ERROR: RELEASE_NAME environment variable was not set"
    fatalError=true
elif [ "$SERVICE_NAME" == "" ]; then
    echo "ERROR: SERVICE_NAME environment variable was not set"
    fatalError=true
elif [ "$NAMESPACE" == "" ]; then
    echo "ERROR: NAMESPACE environment variable was not set"
    fatalError=true
elif [ "$ROUTE_DAEMON" == "true" ]; then
    echo "ERROR: ROUTE_DAEMON environment variable indicates we are running in a daemon set pod"
    fatalError=true
elif ! echo "$HOSTNAME" | grep -q "strongswan"; then
    echo "ERROR: HOSTNAME environment variable indicates we are not running in the VPN strongswan pod"
    fatalError=true
fi
if [ "$fatalError" = true ]; then
    echo
    echo "This tool MUST only be run from within the VPN pod that was deployed by the strongswan helm chart"
    echo "This tool can not be run from a strongswan daemon set pod"
    exit 1
fi

echo "Installing pre-req tools (kubectl)"
install_kubectl
echo "Done"
echo "----------------------------------------------------------------------"

echo "Retrieving the status of the VPN connection:"
sudo /usr/sbin/ipsec statusall | tee /tmp/ipsec.status

if ! grep -q "ESTABLISHED" /tmp/ipsec.status; then
    echo "The VPN connection is not ESTABLISHED"
    echo "----------------------------------------------------------------------"

    if [ "$autoStart" == "start" ]; then
        echo "VPN is configured to start automatically."
        echo "Pinging the remote gateway ($remoteGateway) ..."
        if ! ping -c 3 "$remoteGateway"; then
            echo "WARNING: The remote gateway did not respond to the ping requests"
            echo "CHECK: Is the correct IP address configured for the gateway?"
            echo "CHECK: Is the gateway configured to respond to ping (ICMP requests)?"
            echo "CHECK: Is there a firewall blocking ICMP traffic?"
        fi
        echo "----------------------------------------------------------------------"
    fi

    if [ "$remoteGateway" != "%any" ]; then
        echo "Using nmap to verify the ipsec ports (udp 500/4500) are open"
        sudo /usr/bin/nmap -sU -p 500,4500 -n "$remoteGateway" | tee /tmp/nmap.status
        echo
        if ! grep -q "Host is up" /tmp/nmap.status; then
            echo "WARNING: The remote gateway at $remoteGateway does not appear to be reachable"
        fi
        if grep -q "closed" /tmp/nmap.status; then
            echo "CHECK: Is the on-prem VPN active?"
            echo "CHECK: Is there a firewall blocking udp 500/4500 traffic?"
        fi
        if grep -q "open" /tmp/nmap.status; then
            echo "Looks like IPSec is active on the remote gateway"
            remoteIpSecActive=true
        fi
        echo "----------------------------------------------------------------------"
    fi

    if [ "$autoStart" == "add" ]; then
        echo "The cluster is configured to listen for incoming VPN connection requests"
        if [ "$remoteIpSecActive" = false ]; then
            echo "CHECK: Is the on-prem VPN active?"
        fi
        echo "CHECK: Is the on-prem VPN configured to start the VPN connection?"
        echo
        loadBalancerIp=$(/tmp/kubectl get service -n "$NAMESPACE" "$SERVICE_NAME" -o jsonpath='{ .status.loadBalancer.ingress[0].ip }')
        echo "CHECK: Is the on-prem VPN configured to connect to $loadBalancerIp ?"
        echo "----------------------------------------------------------------------"
    fi

    echo "Retrieving the VPN pod logs and looking for common errors..."
    /tmp/kubectl logs -n "$NAMESPACE" "$(hostname)" | tail -100 >/tmp/pod.log
    echo
    echo "CHECK: For VPN 'state change' messages in the log"
    grep 'state change' /tmp/pod.log
    echo
    echo "CHECK: For 'FAIL' in the log"
    grep -i 'fail' /tmp/pod.log | grep -v 'No such file'
    echo
    echo "CHECK: For 'RETRANSMIT' in the log"
    grep -i 'retransmit' /tmp/pod.log
    echo
    echo "CHECK: For 'NOT RESPONDING' in the log"
    grep -i 'not responding' /tmp/pod.log
    echo "----------------------------------------------------------------------"

    echo "VERIFY the ipsec.conf below matches the on-prem VPN settings:"
    echo
    cat /etc/ipsec.conf
    echo "----------------------------------------------------------------------"

    echo "VERIFY the ipsec.secrets below matches the on-prem VPN settings:"
    echo
    cat /etc/ipsec.secrets
    echo "----------------------------------------------------------------------"

    echo "Finally, the last 100 lines of the VPN pod logs:"
    cat /tmp/pod.log

else
    echo "The VPN connection was ESTABLISHED"
    echo "----------------------------------------------------------------------"

    localSubnet=${localSubnet%?}
    remoteSubnet=${remoteSubnet%?}

    echo "Subnets associated with this VPN configuration:"
    echo
    echo "   Requested local subnets  = $(echo "$localSubnet" | tr ',' ' ')"
    echo "   Requested remote subnets = $(echo "$remoteSubnet" | tr ',' ' ')"
    echo
    actualSubnets=$(tail -1 /tmp/ipsec.status | cut -d':' -f2)
    if echo "$actualSubnets" | grep -q "=="; then
        echo "   Actual local subnets  = $(echo "$actualSubnets" | cut -d'=' -f1 | sed 's/^ *//')"
        echo "   Actual remote subnets = $(echo "$actualSubnets" | cut -d'=' -f4 | sed 's/^ *//')"
        echo
        echo "VERIFY: The requested and actual subnets match"
    else
        echo "   Actual local subnets  = None"
        echo "   Actual remote subnets = None"
        echo
        echo "WARNING: The on-prem VPN endpoint did not accept the local/remote subnets we requested"
        echo "VERIFY: The local and remote subnets are correct"
        echo "CHECK: The on-prem VPN settings and verify that the subnets match"
    fi
    echo "----------------------------------------------------------------------"

    checkCalico=false
    if [ "$ENABLE_POD_SNAT" == "false" ]; then
        checkCalico=true
    elif [ "$ENABLE_POD_SNAT" == "auto" ]; then
        if echo "$localSubnet" | grep -q "172.30.0.0/16"; then
            checkCalico=true
        fi
    fi
    if [ "$checkCalico" = true ]; then
        echo "Listing the calico IPPool resources:"
        echo
        foundMissing=false
        calicoCmd getIPPool | tee /tmp/ippool.output
        IFS=','
        for subnet in $remoteSubnet; do
            if ! grep -q "$subnet" /tmp/ippool.output; then
                echo "WARNING: A calico IPPool resource does not exist for subnet $subnet.  Delete the helm deployment and re-install the helm chart"
                foundMissing=true
            fi
        done
        if [ "$foundMissing" = false ]; then
            echo "VERIFY: There is an IPPool resource for each remote subnet"
        fi
        echo "----------------------------------------------------------------------"
    fi

    echo "Displaying the contents of the routes config map:"
    echo
    /tmp/kubectl get cm -n "$NAMESPACE" "$RELEASE_NAME"-strongswan-routes -o jsonpath='{ .data }' | tr -d '"' | tr '{[ ]},' '\n' | grep ":" | tee /tmp/configmap.routes
    echo
    echo "This config map tells the daemon set what routes need to be created"
    echo "VERIFY: Each field in the 'data:' section is set to the correct value"
    echo "----------------------------------------------------------------------"

    echo "Displaying the status of the VPN pod:"
    /tmp/kubectl get pods -n "$NAMESPACE" -l app=strongswan,release="$RELEASE_NAME" -o wide
    echo
    echo "Displaying the status of the daemon set pods:"
    /tmp/kubectl get pods -n "$NAMESPACE" -l app=strongswan-routes,release="$RELEASE_NAME" -o wide | tee /tmp/daemonset.pods
    echo
    echo "VERIFY: All pods should be: Running"
    echo "----------------------------------------------------------------------"

    echo "Displaying the VPN routes added on each node:"
    echo
    routeTable=$(grep "routeTable:" /tmp/configmap.routes | cut -d":" -f2)
    podList=$(grep -v "^NAME" /tmp/daemonset.pods | awk '{ print $1 }' | tr '\n' ',')
    IFS=','
    for pod in $podList; do
        node=$(grep "$pod" /tmp/daemonset.pods | awk '{ print $7 }')
        echo "VPN routes on $node:"
        /tmp/kubectl exec -n "$NAMESPACE" "$pod" -- sudo /sbin/ip route list table "$routeTable"
        echo
    done
    echo "VERIFY: Routes for each remote subnet have been added to each node"

    echo "----------------------------------------------------------------------"
    echo "Displaying the cross-subnet rules on the VPN node:"
    echo
    workerNodeIP=$(grep "workerNodeIP:" /tmp/configmap.routes | cut -d":" -f2)
    vpnNodeDaemon=$(grep "$workerNodeIP" /tmp/daemonset.pods | awk '{ print $1 }')
    /tmp/kubectl exec -n "$NAMESPACE" "$vpnNodeDaemon" -- sudo /sbin/ip rule list

    echo
    echo "Displaying the cross-subnet routes located in routing table 199:"
    echo
    /tmp/kubectl exec -n "$NAMESPACE" "$vpnNodeDaemon" -- sudo /sbin/ip route list table 199

    if [ -n  "$LOCAL_SUBNET_NAT" ]; then
        echo "----------------------------------------------------------------------"
        echo "Displaying the local subnet NAT table on the VPN node:"
        echo
        sudo /usr/sbin/iptables-legacy --list -t nat
    fi
fi

echo "----------------------------------------------------------------------"
echo "Done"
