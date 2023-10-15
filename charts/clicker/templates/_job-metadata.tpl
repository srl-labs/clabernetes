{{- define "jobMetadata" -}}
labels:
  chart: "{{ .Chart.Name }}-{{ .Chart.Version }}"
  release: {{ .Release.Name }}
  heritage: {{ .Release.Service }}
  clabernetes/app: {{ .Values.appName }}
  clabernetes/name: "{{ .Values.appName }}-clicker"
  clabernetes/component: clicker
  {{- $podLabels := merge .Values.globalLabels .Values.jobLabels }}
    {{- if $podLabels }}
{{ toYaml $podLabels | indent 2 }}
    {{- end }}
{{- $podAnnotations := merge .Values.globalAnnotations .Values.jobAnnotations }}
{{- if $podAnnotations }}
annotations:
{{ toYaml $podAnnotations | indent 2 }}
{{- end }}
{{- end -}}