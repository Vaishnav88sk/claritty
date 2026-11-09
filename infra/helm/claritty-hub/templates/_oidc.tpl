{{- define "claritty-hub.oidcProxy" -}}
{{- $oidc := .Values.edge.oidc -}}
{{- $secret := required "edge.oidc.existingSecret is required" $oidc.existingSecret -}}
{{- $issuer := required "edge.oidc.issuerURL is required" $oidc.issuerURL -}}
{{- if not (hasPrefix "https://" $issuer) -}}
{{- fail "edge.oidc.issuerURL must use HTTPS" -}}
{{- end -}}
{{- if or (not $oidc.allowedEmailDomains) (has "*" $oidc.allowedEmailDomains) -}}
{{- fail "edge.oidc.allowedEmailDomains must list trusted domains explicitly" -}}
{{- end -}}
{{- if or (not $oidc.trustedProxyCIDRs) (has "0.0.0.0/0" $oidc.trustedProxyCIDRs) (has "::/0" $oidc.trustedProxyCIDRs) -}}
{{- fail "edge.oidc.trustedProxyCIDRs must list trusted proxies explicitly" -}}
{{- end -}}
- name: oidc-proxy
  image: {{ required "edge.oidc.image is required" $oidc.image | quote }}
  args:
    - --provider=oidc
    - {{ printf "--oidc-issuer-url=%s" $issuer | quote }}
    - {{ printf "--redirect-url=https://%s/oauth2/callback" .Values.edge.dashboardHost | quote }}
    - --upstream=http://127.0.0.1:8822
    - --http-address=0.0.0.0:4180
    - --reverse-proxy=true
    - --pass-host-header=true
    - --code-challenge-method=S256
    - --insecure-oidc-skip-nonce=false
    - --cookie-secure=true
    - --cookie-samesite=lax
    - --cookie-name=__Host-claritty
    - --cookie-expire=8h
    - --cookie-refresh=1h
    - --skip-provider-button=true
    {{- range $oidc.trustedProxyCIDRs }}
    - {{ printf "--trusted-proxy-ip=%s" . | quote }}
    {{- end }}
    {{- range $oidc.allowedEmailDomains }}
    - {{ printf "--email-domain=%s" . | quote }}
    {{- end }}
    {{- range $oidc.allowedGroups }}
    - {{ printf "--allowed-group=%s" . | quote }}
    {{- end }}
  env:
    {{- range list "CLIENT_ID" "CLIENT_SECRET" "COOKIE_SECRET" }}
    - name: OAUTH2_PROXY_{{ . }}
      valueFrom:
        secretKeyRef:
          name: {{ $secret }}
          key: {{ . }}
    {{- end }}
  ports:
    - name: dashboard
      containerPort: 4180
  readinessProbe:
    httpGet:
      path: /ready
      port: dashboard
  livenessProbe:
    httpGet:
      path: /ping
      port: dashboard
  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop: ["ALL"]
  resources:
    requests:
      cpu: 25m
      memory: 32Mi
    limits:
      cpu: 250m
      memory: 128Mi
{{- end -}}
