{{- define "tags" -}}
{{- $local := dict "first" true -}}
{{- range $k, $v := . -}}{{- if not $local.first -}}{{- ", " -}}{{- end -}}{{- $k | quote | replace "\"" "\\\"" -}}{{- ": " -}}{{- $v | quote | replace "\"" "\\\"" -}}{{- $_ := set $local "first" false -}}{{- end -}}
{{- end -}}