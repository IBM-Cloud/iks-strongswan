{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "strongswan.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "strongswan.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- $release := printf "%s" .Release.Name | trunc 35 | trimSuffix "-" -}}
{{- printf "%s-%s" $release $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
strongSwan image: registry / namespace / name : tag
*/}}
{{- define "strongswan.image" -}}
{{- $tag := printf "%s" .Chart.AppVersion | replace "." "-" -}}
{{- printf "icr.io/armada-master/strongswan:%s" $tag -}}
{{- end -}}

{{/*
strongSwan namespace to use. Force kube-system if RBAC is not enabled
*/}}
{{- define "strongswan.namespace" -}}
    {{- if .Values.enableRBAC }}
        {{- .Release.Namespace -}}
    {{- else -}}
        {{- "kube-system" -}}
    {{- end -}}
{{- end -}}

{{/*
Image pull policy (IfNotPresent or Always)
*/}}
{{- define "strongswan.pullPolicy" -}}
    {{- if .Chart.AppVersion | hasPrefix "dev-" -}}
        {{- "Always" -}}
    {{- else -}}
        {{- "IfNotPresent" -}}
    {{- end -}}
{{- end -}}

{{/*
Kubernetes version - semver style: vX.Y.Z   .Capabilities.KubeVersion = "v1.20.0-beta.1+IKS" or "v1.20.0-rc.0"
*/}}
{{- define "strongswan.kubeVersion" -}}
{{- printf "%s" .Capabilities.KubeVersion | splitList "+" | first | splitList " " | first | trimPrefix "{" -}}
{{- end -}}

{{/*
Determine value that should be used for ipsec.closeaction
*/}}
{{- define "strongswan.closeaction" -}}
    {{- if eq .Values.ipsec.closeaction "auto" -}}
        {{- if eq .Values.ipsec.auto "start" -}}
            {{- "restart" -}}
        {{- else -}}
            {{- "none" -}}
        {{- end -}}
    {{- else -}}
        {{- .Values.ipsec.closeaction -}}
    {{- end -}}
{{- end -}}

{{/*
Determine value that should used if connectUsingLoadBalancerIP==auto
*/}}
{{- define "strongswan.connectUsingLoadBalancerIP" -}}
    {{- if eq .Values.connectUsingLoadBalancerIP "auto" -}}
        {{- if and (eq .Values.ipsec.auto "start") (or .Values.loadBalancerIP .Values.zoneLoadBalancer) -}}
            {{- "true" -}}
        {{- else -}}
            {{- "false" -}}
        {{- end -}}
    {{- else -}}
        {{- .Values.connectUsingLoadBalancerIP -}}
    {{- end -}}
{{- end -}}

{{/*
Retrieve the key of the first item in the nodeSelector: "map[kubernetes.io/hostname:10.184.110.141 strongswan:vpn]"
The second and later items listed in the nodeSelector option are ignored.
*/}}
{{- define "strongswan.nodeSelectorKey" -}}
{{- printf "%s" .Values.nodeSelector | splitList "[" | last | splitList ":" | first -}}
{{- end -}}

{{/*
Retrieve value of the first item in the nodeSelector: "map[kubernetes.io/hostname:10.184.110.141 strongswan:vpn]"
The second and later items listed in the nodeSelector option are ignored.
*/}}
{{- define "strongswan.nodeSelectorValue" -}}
{{- printf "%s" .Values.nodeSelector | splitList "]" | first | splitList " " | first | splitList ":" | last -}}
{{- end -}}
