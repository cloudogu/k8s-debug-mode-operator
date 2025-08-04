{{/* Chart basics
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec) starting from
Kubernetes 1.4+.
*/}}
{{- define "k8s-debug-mode-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/* All-in-one labels */}}
{{- define "k8s-debug-mode-operator.labels" -}}
app: ces
{{ include "k8s-debug-mode-operator.selectorLabels" . }}
helm.sh/chart: {{- printf " %s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
k8s.cloudogu.com/component.name: {{ include "k8s-debug-mode-operator.name" . }}
k8s.cloudogu.com/component.version: {{ .Chart.AppVersion | quote }}
{{- end }}

{{/* Selector labels */}}
{{- define "k8s-debug-mode-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8s-debug-mode-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}