{{/*
展开 chart 全限定名（release-name + chart-name，截断 63 字符）
*/}}
{{- define "tcm.fullName" -}}
{{- if .Values.global.fullNameOverride -}}
{{- .Values.global.fullNameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default "tcm-history-ai" .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
chart 名称
*/}}
{{- define "tcm.name" -}}
{{- default "tcm-history-ai" .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
命名空间（默认取 global.namespace，缺失则用 release namespace）
*/}}
{{- define "tcm.namespace" -}}
{{- default .Release.Namespace .Values.global.namespace -}}
{{- end -}}

{{/*
通用标签：app.kubernetes.io/*
*/}}
{{- define "tcm.labels" -}}
app.kubernetes.io/name: {{ include "tcm.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
按服务生成标签（带 component）
用法：{{ include "tcm.serviceLabels" (list $ "gateway" "api-gateway") }}
*/}}
{{- define "tcm.serviceLabels" -}}
{{- $root := index . 0 -}}
{{- $svc := index . 1 -}}
{{- $component := index . 2 -}}
app.kubernetes.io/name: {{ $svc }}
app.kubernetes.io/instance: {{ $root.Release.Name }}
app.kubernetes.io/version: {{ $root.Chart.AppVersion | quote }}
app.kubernetes.io/component: {{ $component }}
app.kubernetes.io/managed-by: {{ $root.Release.Service }}
{{- end -}}

{{/*
selector labels（按服务）
用法：{{ include "tcm.selectorLabels" (list $ "gateway") }}
*/}}
{{- define "tcm.selectorLabels" -}}
{{- $root := index . 0 -}}
{{- $svc := index . 1 -}}
app.kubernetes.io/name: {{ $svc }}
app.kubernetes.io/instance: {{ $root.Release.Name }}
{{- end -}}

{{/*
镜像全名：registry/service:appVersion
用法：{{ include "tcm.image" (list $ "gateway") }}
*/}}
{{- define "tcm.image" -}}
{{- $root := index . 0 -}}
{{- $svc := index . 1 -}}
{{- printf "%s/%s:%s" $root.Values.global.imageRegistry $svc $root.Values.global.appVersion -}}
{{- end -}}
