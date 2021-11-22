{{/*
Expand the name of the chart.
*/}}
{{- define "liqo-k3s-hollow.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "liqo-k3s-hollow.fullname" -}}
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
{{- define "liqo-k3s-hollow.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "liqo-k3s-hollow.labels" -}}
helm.sh/chart: {{ include "liqo-k3s-hollow.chart" . }}
{{ include "liqo-k3s-hollow.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "liqo-k3s-hollow.selectorLabels" -}}
app.kubernetes.io/name: {{ include "liqo-k3s-hollow.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "liqo-k3s-hollow.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "liqo-k3s-hollow.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Concatenate a values dictionary into a string in the form "key1=val1,key2=val2"
*/}}
{{- define "liqo-k3s-hollow.concatenate" -}}
{{- $res := "" -}}
{{- range $key, $val := ( . | fromYaml ) -}}
{{- $res = print $res $key "=" $val "," -}}
{{- end -}}
{{ trimSuffix "," $res }}
{{- end -}}
