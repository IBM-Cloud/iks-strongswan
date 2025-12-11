# strongSwan IPSec VPN service

![strongSwan IPSec VPN service](https://www.strongswan.org/images/strongswan.png)

## Introduction

You can set up the strongSwan IPSec VPN service to securely connect your Kubernetes cluster with an on-premises network. The strongSwan IPSec VPN service provides a secure end-to-end communication channel over the internet that is based on the industry-standard Internet Protocol Security (IPsec) protocol suite. To set up a secure connection between your cluster and an on-premises network, you must have an IPsec VPN gateway installed in your on-premises data center.

The strongSwan IPSec VPN service is integrated within your Kubernetes cluster.  There is no need to have an external gateway device.  The IPSec VPN endpoint is enabled as a Kubernetes pod/container running in your cluster.  Routes are automatically configured on all of the worker nodes of the cluster when VPN connectivity is established.  These routes can allow two way connectivity from any pod on any worker node through the VPN tunnel to (or from) the remote system.

The strongSwan IPSec VPN service consists of a Kubernetes Load Balancer service, a VPN pod deployment, a VPN routes daemon set, and multiple config maps. All of these resources and bundled together and delivered as a single Kubernetes Helm chart.

You can setup the strongSwan IPSec VPN Service using a Helm chart that allows you to configure and deploy the strongSwan IPSec VPN service inside of a Kubernetes pod.

Because strongSwan is integrated within your cluster, you don't need an external gateway device. When VPN connectivity is established, routes are automatically configured on all of the worker nodes in the cluster. These routes allow two-way connectivity through the VPN tunnel between pods on any worker node and the remote system. For example, the following diagram shows how an app in IBM Cloud Kubernetes Service can communicate with an on-premises server via a strongSwan VPN connection:

![Using Helm to setup strongSwan IPSec VPN service](https://raw.githubusercontent.com/IBM-Bluemix-Docs/containers/master/images/cs_vpn_strongswan.png)

For additional information on how you can use the strongSwan IPsec VPN service to securely connect your worker nodes to your on-premises data center, see [Setting up VPN connectivity](https://cloud.ibm.com/docs/containers?topic=containers-vpn#vpn) in the [IBM Cloud Kubernetes Service](https://www.ibm.com/cloud/container-service) documentation.

## Prerequisites

To set up a secure connection between your cluster and an on-premises network, you must have an IPsec VPN gateway installed in your on-premises data center. There are many ways to get an on-premises IPsec VPN gateway - it can be a hardware device, a Virtual Machine running strongSwan, a Kubernetes Cluster running the strongSwan helm chart.

For additional information on how you can use the strongSwan IPsec VPN service to securely connect your worker nodes to your on-premises data center, see [Setting up VPN connectivity](https://cloud.ibm.com/docs/containers?topic=containers-vpn#vpn) in the [IBM Cloud Kubernetes Service](https://www.ibm.com/cloud/container-service) documentation.

## Resources Required

- This chart will work in any size cluster with any size worker present

## Installing the Chart

- Extract the default configuration settings to a local YAML file. Replace `<repo>` with the IBM Cloud Helm repo name that you added to your Helm installation. For an overview of available repos, see [Adding services by using Helm charts](https://cloud.ibm.com/docs/containers?topic=containers-helm#helm).

    ```bash
    helm inspect values <repo>/strongswan > config.yaml
    ```

- Open the `config.yaml` file and update the default values according to the VPN configuration you want.  If a property has specific values that you can choose from, those values are listed in comments above each property in the file.  See the `Configuration` section for a complete list of all of the configuration values that can be modified

    **Important:** If you do not need to change a property, comment that property out by placing a `#` in front of it.

- Install the Helm Chart to your cluster with the updated config.yaml file

    ```bash
    helm install -f config.yaml vpn <repo>/strongswan
    ```

- Check the chart deployment status

    ```bash
    helm status vpn
    ```

## Chart Details

Specify each parameter that you want to specify in a yaml file and use the  `-f` option with the yaml file name to `helm install`.

The file config.yaml above should specify the various parameters you want to set for your VPN configuration. The cluster where the helm install is done using the above command will have strongSwan setup with the specified configuration.

## Verifying the Chart

- If the VPN on the on-premises gateway is not active, start it

- Check the status of the VPN. A status of `ESTABLISHED` means that the VPN connection was successful.

    ```bash
    export STRONGSWAN_POD=$(kubectl get pod -l app=strongswan,release=vpn -o jsonpath='{ .items[0].metadata.name }')
    kubectl exec $STRONGSWAN_POD -- ipsec status
    ```

    Example output:

    ```bash
    Security Associations (1 up, 0 connecting):
        k8s-conn[1]: ESTABLISHED 17 minutes ago, 172.30.244.42[ibm-cloud]...192.168.253.253[on-prem]
        k8s-conn{2}: INSTALLED, TUNNEL, reqid 12, ESP in UDP SPIs: c78cb6b1_i c5d0d1c3_o
        k8s-conn{2}: 172.21.0.0/16 172.30.0.0/16 === 10.91.152.128/26
    ```

## Validation Tests

Five helm test programs are included with the strongSwan chart.  These test programs can be used to verify the VPN connectivity once the chart has been installed.  Some of these tests  require the `remote.privateIPtoPing` configuration setting to be specified.  Other tests require specific `local.subnet` to be exposed in the VPN configuration or other configuration settings. If those subnets are not exposed, the validation test program will fail.  Depending on the network connectivity requirements, it might be perfectly acceptable for some of these tests to fail.

Use  `helm test` to run the validation test programs:

  ```bash
  $ helm test vpn
  Pod vpn-strongswan-check-config pending
  Pod vpn-strongswan-check-config running
  Pod vpn-strongswan-check-config succeeded
  Pod vpn-strongswan-check-state pending
  Pod vpn-strongswan-check-state running
  Pod vpn-strongswan-check-state succeeded
  Pod vpn-strongswan-ping-remote-gw pending
  Pod vpn-strongswan-ping-remote-gw running
  Pod vpn-strongswan-ping-remote-gw succeeded
  Pod vpn-strongswan-ping-remote-ip-1 pending
  Pod vpn-strongswan-ping-remote-ip-1 running
  Pod vpn-strongswan-ping-remote-ip-1 succeeded
  Pod vpn-strongswan-ping-remote-ip-2 pending
  Pod vpn-strongswan-ping-remote-ip-2 running
  Pod vpn-strongswan-ping-remote-ip-2 succeeded
  .
  .
  ```

NOTES:

- By default, all five of the helm test programs will be run. If you want to run a subset of the helm test programs, specify the helm chart config option: `helmTestsToRun`

The five strongSwan VPN validation test programs are:

1. `vpn-strongswan-check-config` : Performs syntax validation of the ipsec.conf file that is generated from the `config.yaml` file
1. `vpn-strongswan-check-state` : Check to see if the VPN connection is `ESTABLISHED`
1. `vpn-strongswan-ping-remote-gw` : Attempt to ping the `remote.gateway` IP address that was configured in the `config.yaml` file.  It is possible that the VPN could be in `ESTABLISHED` state, but this test case could still fail if ICMP packets are being blocked by a firewall
1. `vpn-strongswan-ping-remote-ip-1` : Attempt to ping the `remote.privateIPtoPing` IP address from a Kubernetes pod.  In order for this test case to pass, the Kubernetes pod subnet, 172.30.0.0/16 needs to be listed in the `local.subnet` property OR `enablePodSNAT` needs to be enabled.
1. `vpn-strongswan-ping-remote-ip-2` : Attempt to ping the `remote.privateIPtoPing` IP address from a worker node.  In order for this test case to pass, the cluster worker node's private subnet needs to be listed in the `local.subnet` property

If any of these test programs fail, the output of the validation test can be viewed by looking at the logs of the test pod:

```bash
kubectl logs  <test program>
```

## Uninstalling the Chart

To uninstall / delete:

  ```bash
  helm uninstall vpn
  ```

## Configuration

The helm chart has the following configuration options:

| Value                        | Description                                       | Default                        |
|------------------------------|---------------------------------------------------|--------------------------------|
| `validate`                   | Type of validation to be done on ipsec.conf       | strict                         |
| `overRideIpsecConf`          | Provide alternative ipsec.conf to use             |                                |
| `overRideIpsecSecrets`       | Provide alternative ipsec.secrets to use          |                                |
| `enablePodSNAT`              | Enable SNAT for pod outbound traffic              | auto                           |
| `enableRBAC`                 | Enable creation of RBAC resources                 | true                           |
| `enableServiceSourceIP`      | Enable externalTrafficPolicy=local on service     | false                          |
| `enableSingleSourceIP`       | Force all Kubernetes traffic to be hidden         | false                          |
| `localNonClusterSubnet`      | Local non cluster subnets to expose over the VPN  |                                |
| `localSubnetNAT`             | Allow NAT of the private local IP subnets         |                                |
| `remoteSubnetNAT`            | Allow NAT of the private remote IP subnets        |                                |
| `loadBalancerIP`             | Public IP address to use for Load Balancer        |                                |
| `zoneLoadBalancer`           | List of Load balancer IP addrs for each zone      |                                |
| `connectUsingLoadBalancerIP` | Connect outbound VPN using LB VIP as source       | auto                           |
| `nodeSelector`               | Kubernetes node selector to apply to VPN pod      |                                |
| `zoneSelector`               | Availability zone for VPN pod and LB service      |                                |
| `zoneSpecificRoutes`         | Limit route config to workers in specified zone   | false                          |
| `privilegedVpnPod`           | Run the VPN pod with privileged authority         | false                          |
| `helmTestsToRun`             | List of tests to run during `helm test`           | ALL                            |
| `tolerations`                | Kubernetes tolerations to apply to daemon set     |                                |
| `strongswanLogging`          | What components log and how much                  | (see file)                     |
| `ipsec.keyexchage`           | Protocol to use with the VPN connection           | ikev2                          |
| `ipsec.esp`                  | ESP encryption/authentication algorithms to use   |                                |
| `ipsec.ike`                  | IKE encryption/authentication algorithms to use   |                                |
| `ipsec.auto`                 | Initiate the VPN connection or listen             | add                            |
| `ipsec.closeaction`          | Action to take if remote peer unexpectedly close  | restart                        |
| `ipsec.dpdaction`            | Controls the use of the DPD protocol              | restart                        |
| `ipsec.ikelifetime`          | How long keying channel should last               | 3h                             |
| `ipsec.keyingtries`          | How many attempts to negotiate a connection       | %forever                       |
| `ipsec.lifetime`             | How long should instance of connection last       | 1h                             |
| `ipsec.margintime`           | How long before expiry should re-negotiate begin  | 9m                             |
| `ipsec.additionalOptions`    | Any additional ipsec.conf options desired         |                                |
| `local.subnet`               | Local subnet(s) exposed over the VPN              | 172.30.0.0/16,172.21.0.0/16    |
| `local.zoneSubnet`           | Zone specific local subnet(s) for the VPN         |                                |
| `local.id`                   | String identifier for the local side              | ibm-cloud                      |
| `remote.gateway`             | IP address of the on-prem VPN gateway             | %any                           |
| `remote.subnet`              | Remote subnets to access from the cluster         | 192.168.0.0/24                 |
| `remote.id`                  | String identifier for the remote side             | on-prem                        |
| `remote.privateIPtoPing`     | IP address in the remote subnet to use for tests  |                                |
| `preshared.secret`           | Pre-shared secret.  Stored in ipsec.secrets       | "strongswan-preshared-secret"  |
| `monitoring.enable`          | Enable monitoring for the VPN connection          | false                          |
| `monitoring.clusterName`     | Name of Kubernetes cluster                        |                                |
| `monitoring.privateIPs`      | IP(s) for monitoring to ping                      |                                |
| `monitoring.httpEndpoints`   | HTTP endpoint(s) for monitoring to test           |                                |
| `monitoring.timeout`         | Time that a monitoring test must complete in      | 5 (seconds)                    |
| `monitoring.delay`           | Time delay between monitoring test cycles         | 120 (seconds)                  |
| `monitoring.slackWebhook`    | Slack Webhook URL from Incoming Webhooks config   |                                |
| `monitoring.slackChannel`    | Slack channel to post monitoring results          |                                |
| `monitoring.slackUsername`   | Slack username to associate with monitor messages | "IBM strongSwan VPN"           |
| `monitoring.slackIcon`       | Slack icon to associate with monitor messages     | ":swan:"                       |

## Slack Configuration

The strongSwan VPN can be configured to automatically post VPN connectivity messages to a Slack channel.
This is accomplished using the Slack Custom Integrations **Incoming WebHooks** app.

Before strongSwan VPN can be configured to send Slack notifications, the **Incoming WebHooks** app needs to be installed into your Slack workspace and configured:

- install this app into your workspace by going to <https://slack.com/apps/A0F7XDUAZ-incoming-webhooks> and clicking **Request to Install**.  If this app is not listed in your Slack setup, contact your slack administrator.
- once your request to install has been approved, click **Add Configuration** to create a new configuration
- on the **Post to Channel** drop down box, select an existing Slack channel -OR- create a new Slack channel
- once the new configuration has been created, a unique **Webhook URL** will be generated.  The Webhook URL should look something like: <https://hooks.slack.com/services/.../.../...>

This unique **Webhook URL** needs to be specified in your strongSwan VPN helm configuration file.
The strongSwan logic will automatically post VPN connectivity message to this **Webhook URL** which will add the messages to the Slack channel that you configured.

In order to verify that strongSwan VPN slack notifications will work, the following commands will send a test message to your **Webhook URL**:

```bash
export WEBHOOK_URL="<PUT YOUR WEBHOOK URL HERE>"
curl -X POST -H 'Content-type: application/json' -d '{"text":"VPN test message"}' $WEBHOOK_URL
```

It is important to verify that the test message was successful.

## Troubleshooting

To help resolve some common VPN configuration issues, a `vpnDebug` tool has been created and is delivered with the strongSwan image.  To run this debug tool:

```bash
export STRONGSWAN_POD=$(kubectl get pod -l app=strongswan,release=vpn -o jsonpath='{ .items[0].metadata.name }')`
kubectl exec $STRONGSWAN_POD -- vpnDebug
```

The tool will dump out several pages of information as it runs various tests trying to determine common networking issues.  Output lines that begin with: `ERROR`, `WARNING`, `VERIFY`, or `CHECK` indicate possible errors with the VPN connectivity.

## Limitations

There are a few scenarios in which strongSwan helm chart may not be the best choice:

- The strongSwan helm chart does not support "route based" IPSec VPNs. If a "route based" IPSec VPN is required, a different VPN solution will need to be used.
- The strongSwan helm chart only supports IPSec VPNs using "preshared keys". If certificates are required, a a different VPN solution will need to be used.
- The strongSwan helm chart is not a general purpose VPN gateway solution. It is not designed to allow multiple clusters and other IaaS resources to share a single VPN connection. If multiple clusters need to share a single VPN connection, a different VPN solution should be considered.
- The strongSwan helm chart runs as a Kubernetes pod inside of the cluster. As a result, the performance of the VPN will be affected by the memory and network usage of Kubernetes and other pods that are running in the cluster. In performance critical environments, a VPN solution running outside of the cluster on dedicated hardware should be considered.
- The strongSwan helm chart does not provide any metrics or monitoring of the network traffic flowing over the VPN connection. Other Kubernetes tools should be used for this purpose.
- The strongSwan helm chart runs a single VPN pod as the IPSec tunnel endpoint. Kubernetes will restart the pod if it fails, but there will be a slight down time while the new pod starts up and the VPN connection is re-established. If faster error recovery or a more elaborate HA solution is required, a different VPN solution should be considered.

## Security fixes

| CVE             | Version |
|-----------------|---------|
| CVE-2024-6874   | 2.9.5   |
| CVE-2024-6197   | 2.9.5   |
| CVE-2024-5535   | 2.9.5   |
| CVE-2024-7264   | 2.9.6   |
| CVE-2024-6119   | 2.9.6   |
| CVE-2024-8096   | 2.9.7   |
| CVE-2024-8096   | 2.9.9   |
| CVE-2024-9143   | 2.9.9   |
| CVE-2024-9681   | 2.9.10  |
| CVE-2024-11053  | 2.10.0  |
| CVE-2024-45338  | 2.10.0  |
| CVE-2024-13176  | 2.10.3  |
| CVE-2024-12797  | 2.10.3  |
| CVE-2025-26519  | 2.10.3  |
| CVE-2025-0725   | 2.10.3  |
| CVE-2025-0665   | 2.10.3  |
| CVE-2025-0167   | 2.10.3  |
| CVE-2025-29087  | 2.10.4  |
| CVE-2025-31498  | 2.10.4  |
| CVE-2025-22872  | 2.10.5  |
| CVE-2025-32462  | 2.10.6  |
| CVE-2025-32463  | 2.10.6  |
| CVE-2025-4575   | 2.10.6  |
| CVE-2025-6965   | 2.10.7  |


## Version History

| Version | Date       | Description                                             | Image Tag | Status       |
|---------|------------|---------------------------------------------------------|-----------|--------------|
| 1.0.0   | 12/06/2017 | Initial prod version of the helm chart                  | 17-12-05  | Unsupported  |
| 1.0.8   |  1/10/2018 | Helm test programs, vpnDebug tool, misc fixes           | 18-01-10  | Unsupported  |
| 1.1.0   |  2/05/2018 | Allow on-prem to ping cross subnet worker node          | 18-02-05  | Unsupported  |
| 1.1.1   |  2/07/2018 | Allow SNAT for pod traffic to on-prem network           | 18-02-06  | Unsupported  |
| 1.1.2   |  2/08/2018 | Allow config of tolerations for VPN route daemon        | 18-02-06  | Unsupported  |
| 1.1.3   |  2/08/2018 | Don't delete rule if routes still in table              | 18-02-08  | Unsupported  |
| 1.1.4   |  2/08/2018 | Add anti-affinity to the VPN pod                        | 18-02-08  | Unsupported  |
| 1.1.5   |  2/12/2018 | Add support for local subnet NAT                        | 18-02-11  | Unsupported  |
| 1.1.6   |  2/14/2018 | Expose more ipsec.conf properties in helm chart         | 18-02-11  | Unsupported  |
| 1.1.7   |  2/17/2018 | Change helm ping tests to fail if remote IP not config  | 18-02-11  | Unsupported  |
| 2.0.0   |  2/18/2018 | Prod version - 2.0.0                                    | 18-02-11  | Unsupported  |
| 2.0.1   |  2/21/2018 | Do not remove routes if VPN is still active             | 18-02-21  | Unsupported  |
| 2.0.2   |  2/25/2018 | Non-buffered logging and ability to config logging      | 18-02-22  | Unsupported  |
| 2.0.3   |  2/25/2018 | Always use non-buffered logging                         | 18-02-22  | Unsupported  |
| 2.0.4   |  3/02/2018 | Add heritage=tiller label, fix helm test                | 18-02-22  | Unsupported  |
| 2.0.5   |  3/06/2018 | Change default for ipsec.closeaction                    | 18-02-22  | Unsupported  |
| 2.0.6   |  3/12/2018 | Add arch node affinity                                  | 18-02-22  | Unsupported  |
| 2.0.7   |  3/14/2018 | Update chart description, add appVersion                | 18-02-22  | Unsupported  |
| 2.0.8   |  3/21/2018 | Add rbac support                                        | 18-03-20  | Unsupported  |
| 2.0.9   |  3/28/2018 | Allow outbound VPN connection over LB VIP               | 18-03-28  | Unsupported  |
| 2.0.10  |  4/13/2018 | Add enableServiceSourceIP config, IKEv1 validation      | 18-04-13  | Unsupported  |
| 2.0.11  |  4/20/2018 | Add support for Calico V3                               | 18-04-20  | Unsupported  |
| 2.0.12  |  4/23/2018 | Add support for running in a different namespace        | 18-04-23  | Unsupported  |
| 2.0.13  |  4/25/2018 | New privilegedVpnPod config option                      | 18-04-25  | Unsupported  |
| 2.1.0   |  4/26/2018 | Prod version - 2.1.0                                    | 18-04-25  | Unsupported  |
| 2.1.1   |  5/01/2018 | New enableSingleSourceIP config option                  | 18-05-01  | Unsupported  |
| 2.1.2   |  5/22/2018 | Remove env variable passed to VPN pod                   | 18-05-22  | Unsupported  |
| 2.1.3   |  5/29/2018 | New remoteSubnetNAT config option                       | 18-05-29  | Unsupported  |
| 2.1.4   |  5/31/2018 | New localNonClusterSubnet config option                 | 18-05-31  | Unsupported  |
| 2.1.5   |  6/02/2018 | Fix enablePodSNAT:auto when pod subnet not 172.30.0.0   | 18-06-02  | Unsupported  |
| 2.1.6   |  6/04/2018 | Fix IPPool create/delete during Kube 1.10 upgrade       | 18-06-04  | Unsupported  |
| 2.1.7   |  6/06/2018 | Simplify IPPool delete logic in VPN pod cleanup         | 18-06-06  | Unsupported  |
| 2.1.8   |  6/08/2018 | New enableRBAC config option                            | 18-06-06  | Unsupported  |
| 2.1.9   |  6/18/2018 | Refactor strongswan GO logic                            | 18-06-18  | Unsupported  |
| 2.2.0   |  6/29/2018 | Prod version - 2.2.0                                    | 18-06-18  | Unsupported  |
| 2.2.1   |  7/02/2018 | Allow multi-line and spaces in list values.yaml fields  | 18-06-18  | Unsupported  |
| 2.2.2   |  7/11/2018 | Add alpine license files to chart                       | 18-06-18  | Unsupported  |
| 2.2.3   |  7/26/2018 | Fix validation bug: privateIPtoPing in rightsubnet      | 18-07-26  | Unsupported  |
| 2.2.4   |  8/10/2018 | Add flags to enable just the tests you need             | 18-07-26  | Unsupported  |
| 2.2.5   |  9/11/2018 | Fix error during helm install if validate="off"         | 18-07-26  | Unsupported  |
| 2.2.6   |  9/26/2018 | Prevent LB IP pending when specific LB IP requested     | 18-09-26  | Unsupported  |
| 2.2.7   | 10/29/2018 | Adding Monitoring to VPN Tunnel                         | 18-10-29  | Unsupported  |
| 2.2.8   | 10/30/2018 | Improve create calico IPPool failure message            | 18-10-30  | Unsupported  |
| 2.2.9   | 10/31/2018 | Add resource requests, priorityClassName, apps/v1       | 18-10-30  | Unsupported  |
| 2.3.0   | 11/05/2018 | Prod version - 2.3.0                                    | 18-10-30  | Unsupported  |
| 2.3.1   | 11/20/2018 | Add support for MZR clusters                            | 18-11-20  | Unsupported  |
| 2.3.2   | 12/04/2018 | Add readinessProbe, livenessProbe, and other misc fixes | 18-12-04  | Unsupported  |
| 2.3.3   |  1/04/2019 | Update to alpine 3.8 and GO 1.11.4                      | 19-01-04  | Unsupported  |
| 2.3.4   |  1/17/2019 | New local.id options: %nodePublicIP and %loadBalancerIP | 19-01-17  | Unsupported  |
| 2.3.5   |  1/29/2019 | New zoneLoadBalancer config option                      | 19-01-24  | Unsupported  |
| 2.4.0   |  1/31/2019 | Prod version - 2.4.0                                    | 19-01-24  | Unsupported  |
| 2.4.1   |  2/05/2019 | Fix nodeSelector logic, require helm 2.8+               | 19-01-24  | Unsupported  |
| 2.4.2   |  2/19/2019 | Update to GO 1.11.5                                     | 19-02-19  | Unsupported  |
| 2.4.3   |  3/18/2019 | Update to alpine 3.9 and GO 1.12.1                      | 19-03-18  | Unsupported  |
| 2.5.0   |  3/26/2019 | Prod version - 2.5.0                                    | 19-03-18  | Unsupported  |
| 2.5.1   |  6/10/2019 | Security fixes, conn tracking fix, switch to GO mod     | 19-06-10  | Unsupported  |
| 2.5.2   |  6/13/2019 | New local.zoneSubnet option                             | 19-06-13  | Unsupported  |
| 2.5.3   |  6/20/2019 | Fixed nodeSelectorValue generation                      | 19-06-13  | Unsupported  |
| 2.5.4   |  8/15/2019 | Security fixes, update values.yaml doc                  | 19-08-15  | Unsupported  |
| 2.5.5   |  9/05/2019 | Security fixes and GO 1.12.9                            | 19-09-05  | Unsupported  |
| 2.5.6   | 10/24/2019 | Security fixes, GO 1.12.10, and bug fix                 | 19-10-24  | Unsupported  |
| 2.5.7   | 11/20/2019 | Security fixes, GO 1.12.11, and Alpine 3.10             | 19-11-20  | Unsupported  |
| 2.5.8   | 12/20/2019 | Security fixes                                          | 19-12-20  | Unsupported  |
| 2.6.0   |  1/15/2020 | Security fixes, Helm 3.x, Alpine 3.11, GO 1.13.5        | 20-01-15  | Unsupported  |
| 2.6.1   |  2/02/2020 | Update calicoctl, support Calico KDD                    | 20-02-02  | Unsupported  |
| 2.6.2   |  4/28/2020 | Security fixes                                          | 20-04-28  | Unsupported  |
| 2.6.3   |  5/14/2020 | Security fixes                                          | 20-05-15  | Unsupported  |
| 2.6.4   |  7/16/2020 | Security fixes, GO 1.13.12, and Alpine 3.12             | 20-07-20  | Unsupported  |
| 2.6.5   |  8/14/2020 | Update routes config map                                | 20-07-20  | Unsupported  |
| 2.6.6   | 10/22/2020 | Security fixes, GO 1.15, bug fix calico ippool cleanup  | 20-10-22  | Unsupported  |
| 2.6.7   | 12/10/2020 | Security fixes, Container Provenance                    | 20-12-10  | Unsupported  |
| 2.6.8   | 12/18/2020 | Security fixes, and bug fix in VPN Debug script         | 20-12-18  | Unsupported  |
| 2.6.9   |  1/12/2021 | Pushing to internal Docker registry                     | 21-01-12  | Unsupported  |
| 2.6.10  |  1/26/2021 | Update helm tests to not use Alpine image               | 21-01-26  | Unsupported  |
| 2.7.0   |  2/15/2021 | Running Strongswan as non-root user                     | 21-02-19  | Unsupported  |
| 2.7.1   |  2/19/2021 | Security Fixes                                          | 21-02-20  | Unsupported  |
| 2.7.2   |  4/15/2021 | Security Fixes, GO 1.15.10                              | 21-04-10  | Unsupported  |
| 2.7.3   |  4/20/2021 | Security Fixes, GO 1.15.11                              | 21-04-30  | Unsupported  |
| 2.7.4   |  6/07/2021 | Security Fixes, GO 1.16.4                               | 21-06-10  | Unsupported  |
| 2.7.5   |  8/05/2021 | Security Fixes, GO 1.16.6                               | 21-08-06  | Unsupported  |
| 2.7.6   |  9/09/2021 | Security Fixes, GO 1.16.7                               | 21-09-09  | Unsupported  |
| 2.7.7   |  9/14/2021 | Openshift 4.8 helm test failure                         | 21-09-09  | Unsupported  |
| 2.7.8   |  9/16/2021 | Alpine 3.14 (strongSwan 5.8.4 to 5.9.1)                 | 21-09-16  | Unsupported  |
| 2.7.9   |  9/20/2021 | Remove arch node affinity, change zone node label       | 21-09-20  | Unsupported  |
| 2.7.10  | 11/03/2021 | Security Fixes, GO 1.16.9                               | 21-11-05  | Unsupported  |
| 2.7.11  | 12/12/2021 | Security Fixes, GO 1.17.4, Alpine 3.15                  | 21-12-12  | Unsupported  |
| 2.7.12  |  2/01/2022 | Security Fixes, GO 1.17.6                               | 22-02-01  | Unsupported  |
| 2.7.13  |  3/25/2022 | Security Fixes, GO 1.17.8                               | 22-03-25  | Unsupported  |
| 2.7.14  |  5/02/2022 | Security Fixes, GO 1.17.9                               | 22-05-05  | Unsupported  |
| 2.7.15  |  7/15/2022 | Security Fixes, GO 1.17.11                              | 22-07-15  | Unsupported  |
| 2.7.16  |  8/17/2022 | Security Fixes, GO 1.17.13                              | 22-08-20  | Unsupported  |
| 2.7.17  |  8/19/2022 | Calico 3.23.x IPPool RBAC fix                           | 22-08-20  | Unsupported  |
| 2.7.18  | 10/04/2022 | Security Fixes, GO 1.19.1                               | 22-10-04  | Unsupported  |
| 2.7.19  | 10/12/2022 | Security Fixes                                          | 22-10-12  | Unsupported  |
| 2.7.20  | 11/2/2022  | Security Fixes                                          | 22-11-05  | Unsupported  |
| 2.7.21  | 12/12/2022 | Remove support for Calico Etcd secret                   | 22-12-12  | Unsupported  |
| 2.8.0   |  1/18/2023 | Security Fixes, GO 1.19.5, Alpine 3.17                  | 23-01-25  | Unsupported  |
| 2.8.1   |  2/22/2023 | Security Fixes                                          | 23-02-22  | Unsupported  |
| 2.8.2   |  3/15/2023 | Security Fixes                                          | 23-03-17  | Unsupported  |
| 2.8.3   |  3/31/2023 | Security Fixes                                          | 23-04-05  | Unsupported  |
| 2.8.4   |  5/30/2023 | Security Fixes, GO 1.19.9, Alpine 3.18                  | 23-06-05  | Unsupported  |
| 2.8.5   |  8/25/2023 | Security Fixes, GO 1.20.7                               | 23-08-28  | Unsupported  |
| 2.8.6   |  9/28/2023 | Security Fixes, GO 1.20.8                               | 23-10-04  | Unsupported  |
| 2.8.7   | 10/19/2023 | Security Fixes, GO 1.20.10                              | 23-10-24  | Unsupported  |
| 2.8.8   | 12/01/2023 | Security Fixes                                          | 23-12-05  | Unsupported  |
| 2.8.9   | 12/12/2023 | Security Fixes, GO 1.20.12, Alpine 3.19                 | 23-12-12  | Unsupported  |
| 2.9.0   | 01/08/2024 | Security Fixes, GO 1.21.5, kubectl 1.23.17              | 24-01-11  | Unsupported  |
| 2.9.1   | 02/16/2024 | Security Fixes, GO 1.21.7                               | 24-02-21  | Unsupported  |
| 2.9.2   | 04/12/2024 | Security Fixes, GO 1.21.9                               | 24-04-16  | Unsupported  |
| 2.9.3   | 05/28/2024 | Security Fixes, GO 1.22.3, Alpine 3.20                  | 24-06-03  | Unsupported  |
| 2.9.4   | 06/18/2024 | Security Fixes                                          | 24-06-20  | Unsupported  |
| 2.9.5   | 08/26/2024 | Security Fixes, G0 1.22.6, kubectl 1.25.16              | 24-08-29  | Deprecated   |
| 2.9.6   | 09/10/2024 | Security Fixes, G0 1.22.7                               | 24-09-12  | Deprecated   |
| 2.9.7   | 10/01/2024 | Security Fixes, Fix Broken `kubectl` Download           | 24-10-01  | Deprecated   |
| 2.9.8   | 10/01/2024 | Fix Incorrect `kubectl` URL                             | 24-10-02  | Deprecated   |
| 2.9.9   | 10/21/2024 | Security Fixes                                          | 24-10-23  | Deprecated   |
| 2.9.10  | 12/06/2024 | Security Fixes, GO 1.22.10                              | 24-12-10  | Deprecated   |
| 2.10.0  | 01/08/2025 | Security Fixes, Alpine 3.21                             | 25-01-09  | Deprecated   |
| 2.10.1  | 01/09/2025 | Restore `sudo` for `iptables` Binary                    | 25-01-10  | Deprecated   |
| 2.10.2  | 01/10/2025 | Correct Path of `iptables` Binary                       | 25-01-11  | Deprecated   |
| 2.10.3  | 02/14/2025 | Security Fixes, GO 1.23.6                               | 25-02-19  | Supported    |
| 2.10.4  | 04/21/2025 | Security Fixes, GO 1.23.8                               | 25-04-30  | Supported    |
| 2.10.5  | 06/19/2025 | Security Fixes, GO 1.24.4, Alpine 3.22                  | 25-06-30  | Supported    |
| 2.10.6  | 07/09/2025 | Security Fixes, GO 1.24.5                               | 25-07-18  | Supported    |
| 2.10.7  | 08/11/2025 | Security Fixes, GO 1.24.6                               | 25-08-20  | Recommended  |


- The strongSwan chart v2.8.6 was tested on Helm CLI v3.12.3.
- The strongSwan chart v2.8.7 was tested on Helm CLI v3.13.1.
- The strongSwan chart v2.8.8 was tested on Helm CLI v3.13.2.
- The strongSwan chart v2.8.9 was tested on Helm CLI v3.13.2.
- The strongSwan chart v2.9.0 was tested on Helm CLI v3.13.3.
- The strongSwan chart v2.9.1 was tested on Helm CLI v3.14.1.
- The strongSwan chart v2.9.2 was tested on Helm CLI v3.14.3.
- The strongSwan chart v2.9.3 was tested on Helm CLI v3.15.1.
- The strongSwan chart v2.9.4 was tested on Helm CLI v3.15.2.
- The strongSwan chart v2.9.5 was tested on Helm CLI v3.15.4.
- The strongSwan chart v2.9.6 was tested on Helm CLI v3.15.4.
- The strongSwan chart v2.9.7 was tested on Helm CLI v3.16.1.
- The strongSwan chart v2.9.8 was tested on Helm CLI v3.16.1.
- The strongSwan chart v2.9.9 was tested on Helm CLI v3.16.2.
- The strongSwan chart v2.9.10 was tested on Helm CLI v3.16.3.
- The strongSwan chart v2.10.0 was tested on Helm CLI v3.16.4.
- The strongSwan chart v2.10.1 was tested on Helm CLI v3.16.4.
- The strongSwan chart v2.10.2 was tested on Helm CLI v3.16.4.
- The strongSwan chart v2.10.3 was tested on Helm CLI v3.17.1.
- The strongSwan chart v2.10.4 was tested on Helm CLI v3.17.3.
- The strongSwan chart v2.10.5 was tested on Helm CLI v3.18.3.
- The strongSwan chart v2.10.5 was tested on Helm CLI v3.18.4.
- The strongSwan chart v2.10.6 was tested on Helm CLI v3.18.4.
- The strongSwan chart v2.10.7 was tested on Helm CLI v3.18.4.

A majority of the strongSwan updates over the past 2+ years have been the result of picking up required security fixes.
As a result, the following changes in strongSwan support are being implemented:

- the latest strongSwan chart is `Recommended`. It supports all current releases of IKS
- charts that have been released in the last 6 months are `Supported`
- charts older than 6 months, but less than a year are `Deprecated` and will no longer be supported soon
- charts older than 1 year are `Unsupported`.  Please upgrade to latest release
