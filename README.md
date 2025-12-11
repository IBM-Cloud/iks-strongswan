<!-- markdownlint-disable MD013 -->
# strongSwan IPSec VPN service

![strongSwan IPSec VPN service](https://www.strongswan.org/images/strongswan.png)

## Introduction

You can set up the strongSwan IPSec VPN service to securely connect your Kubernetes cluster with an on-premises network. The strongSwan IPSec VPN service provides a secure end-to-end communication channel over the internet that is based on the industry-standard Internet Protocol Security (IPsec) protocol suite. To set up a secure connection between your cluster and an on-premises network, you must have an IPsec VPN gateway installed in your on-premises data center.

The strongSwan IPSec VPN service is integrated within your Kubernetes cluster.  There is no need to have an external gateway device.  The IPSec VPN endpoint is enabled as a Kubernetes pod/container running in your cluster.  Routes are automatically configured on all of the worker nodes of the cluster when VPN connectivity is established.  These routes can allow two way connectivity from any pod on any worker node through the VPN tunnel to (or from) the remote system.

The strongSwan IPSec VPN service consists of a Kubernetes Load Balancer service, a VPN pod deployment, a VPN routes daemon set, and multiple config maps. All of these resources and bundled together and delivered as a single Kubernetes Helm chart.

You can setup the strongSwan IPSec VPN Service using a Helm chart that allows you to configure and deploy the strongSwan IPSec VPN service inside of a Kubernetes pod.

Because strongSwan is integrated within your cluster, you don't need an external gateway device. When VPN connectivity is established, routes are automatically configured on all of the worker nodes in the cluster. These routes allow two-way connectivity through the VPN tunnel between pods on any worker node and the remote system.
