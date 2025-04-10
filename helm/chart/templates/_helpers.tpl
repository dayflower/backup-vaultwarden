{{/*
Expand the name of the chart.
*/}}
{{- define "backup-vaultwarden.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "backup-vaultwarden.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "backup-vaultwarden.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "backup-vaultwarden.labels" -}}
helm.sh/chart: {{ include "backup-vaultwarden.chart" . }}
{{ include "backup-vaultwarden.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "backup-vaultwarden.selectorLabels" -}}
app.kubernetes.io/name: {{ include "backup-vaultwarden.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "backup-vaultwarden.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "backup-vaultwarden.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}



{{/*
Rclone config file mount path
*/}}
{{- define "backup-vaultwarden.rCloneConfMountPath" -}}
{{- if .Values.rclone.destination }}
{{- default "/etc/rclone" .Values.rclone.conf.mountPath }}
{{- end }}
{{- end }}

{{/*
Rclone source config file mount path
*/}}
{{- define "backup-vaultwarden.rCloneSourceConfMountPath" -}}
{{- if .Values.rclone.destination }}
{{- default "/etc/default/rclone" .Values.rclone.sourceConf.mountPath }}
{{- end }}
{{- end }}

{{/*
Rclone config file name
*/}}
{{- define "backup-vaultwarden.rCloneConfFileName" -}}
{{- if .Values.rclone.destination }}
{{- default "rclone.conf" .Values.rclone.conf.fileName }}
{{- end }}
{{- end }}

{{/*
Rclone config file full path
*/}}
{{- define "backup-vaultwarden.rCloneConfFullFileName" -}}
{{- if .Values.rclone.destination }}
{{- printf "%s/%s" (include "backup-vaultwarden.rCloneConfMountPath" .) (include "backup-vaultwarden.rCloneConfFileName" .) }}
{{- end }}
{{- end }}

{{/*
Rclone config source file full path
*/}}
{{- define "backup-vaultwarden.rCloneSourceConfFullFileName" -}}
{{- if .Values.rclone.destination }}
{{- printf "%s/%s" (include "backup-vaultwarden.rCloneSourceConfMountPath" .) (include "backup-vaultwarden.rCloneConfFileName" .) }}
{{- end }}
{{- end }}
