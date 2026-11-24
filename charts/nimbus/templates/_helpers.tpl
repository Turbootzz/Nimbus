{{- define "nimbus.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "nimbus.fullname" -}}
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

{{- define "nimbus.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nimbus.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "nimbus.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{ include "nimbus.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "nimbus.secretName" -}}
{{- default (include "nimbus.fullname" .) .Values.secrets.existingSecret }}
{{- end }}

{{/* Shorten the base, not the suffix, so it never collides with the app Service */}}
{{- define "nimbus.postgresql.fullname" -}}
{{- printf "%s-postgresql" (include "nimbus.fullname" . | trunc 52 | trimSuffix "-") }}
{{- end }}

{{/* The built-in database always uses nimbus/nimbus, like docker-compose.yml */}}
{{- define "nimbus.databaseHost" -}}
{{- if .Values.postgresql.enabled }}
{{- include "nimbus.postgresql.fullname" . }}
{{- else }}
{{- required "externalDatabase.host is required when postgresql.enabled is false" .Values.externalDatabase.host }}
{{- end }}
{{- end }}
