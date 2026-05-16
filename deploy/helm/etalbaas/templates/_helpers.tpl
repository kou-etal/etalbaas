{{/*
EtalBaaS Helm Chart - Template Helpers
*/}}

{{/*
Chart full name (release-name truncated to 63 chars).
*/}}
{{- define "etalbaas.fullname" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Platform namespace.
*/}}
{{- define "etalbaas.namespace" -}}
{{- .Values.global.platformNamespace }}
{{- end }}

{{/*
Base domain.
*/}}
{{- define "etalbaas.baseDomain" -}}
{{- .Values.global.baseDomain }}
{{- end }}

{{/*
Common labels applied to all resources.
*/}}
{{- define "etalbaas.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: etalbaas
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end }}

{{/*
Selector labels (subset for matchLabels).
*/}}
{{- define "etalbaas.selectorLabels" -}}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service-specific labels. Pass a dict with "root" (.) and "component" (string).
Usage: {{ include "etalbaas.serviceLabels" (dict "root" . "component" "tenant-user") }}
*/}}
{{- define "etalbaas.serviceLabels" -}}
app.kubernetes.io/name: {{ .component }}
app.kubernetes.io/component: {{ .component }}
{{ include "etalbaas.labels" .root }}
{{ include "etalbaas.selectorLabels" .root }}
{{- end }}

{{/*
Resolve full image reference for a service.
Pass a dict with "root" (.), "svc" (values key like "tenantUser"), "name" (image name).
Usage: {{ include "etalbaas.imageRef" (dict "root" . "svc" "tenantUser" "name" "etalbaas-tenant-user") }}
*/}}
{{- define "etalbaas.imageRef" -}}
{{- $svcValues := index .root.Values.services .svc -}}
{{- $repo := $svcValues.image.repository -}}
{{- $tag := $svcValues.image.tag | default "latest" -}}
{{- if $repo -}}
{{ $repo }}:{{ $tag }}
{{- else -}}
{{ .root.Values.global.imageRegistry }}/{{ .name }}:{{ $tag }}
{{- end -}}
{{- end }}

{{/*
Resolve full image reference for the operator.
*/}}
{{- define "etalbaas.operatorImageRef" -}}
{{- $repo := .Values.operator.image.repository -}}
{{- $tag := .Values.operator.image.tag | default "latest" -}}
{{- if $repo -}}
{{ $repo }}:{{ $tag }}
{{- else -}}
{{ .Values.global.imageRegistry }}/etalbaas-operator:{{ $tag }}
{{- end -}}
{{- end }}

{{/*
Common environment variables shared by all microservices.
Provides PORT, METRICS_PORT, ENV, OTEL endpoint, DATABASE_URL.
*/}}
{{- define "etalbaas.commonEnv" -}}
- name: PORT
  value: {{ .Values.services.common.port | quote }}
- name: METRICS_PORT
  value: {{ .Values.services.common.metricsPort | quote }}
- name: ENV
  value: {{ .Values.global.environment | quote }}
- name: OTEL_EXPORTER_OTLP_ENDPOINT
  value: {{ .Values.services.common.otelEndpoint | quote }}
- name: DATABASE_URL
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secrets.platformSecretName }}
      key: DATABASE_URL
{{- end }}

{{/*
Pod-level security context (applied to spec.securityContext).
*/}}
{{- define "etalbaas.podSecurityContext" -}}
runAsNonRoot: true
seccompProfile:
  type: RuntimeDefault
{{- end }}

{{/*
Container-level security context.
*/}}
{{- define "etalbaas.containerSecurityContext" -}}
runAsNonRoot: true
runAsUser: 65532
runAsGroup: 65532
readOnlyRootFilesystem: true
allowPrivilegeEscalation: false
capabilities:
  drop:
    - ALL
{{- end }}
