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
镜像全名：registry/service:appVersion 或 registry/service@digest
用法：{{ include "tcm.image" (list $ "gateway") }}

行为：
  - 若 global.imageDigest 非空：输出 <registry>/<svc>@<digest>（生产推荐，digest 不可变）
  - 否则：输出 <registry>/<svc>:<appVersion>（默认，tag 可变但版本号固定）

生产建议：CI 在镜像构建后通过 --set global.imageDigest=sha256:... 注入 digest，
覆盖 tag 可变性，保证可追溯与可回滚。
*/}}
{{- define "tcm.image" -}}
{{- $root := index . 0 -}}
{{- $svc := index . 1 -}}
{{- if $root.Values.global.imageDigest -}}
{{- printf "%s/%s@%s" $root.Values.global.imageRegistry $svc $root.Values.global.imageDigest -}}
{{- else -}}
{{- printf "%s/%s:%s" $root.Values.global.imageRegistry $svc $root.Values.global.appVersion -}}
{{- end -}}
{{- end -}}
