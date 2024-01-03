apiVersion: v1
kind: Pod
spec:
  containers:
    - name: {{ .SidecarName }}
      image: {{ .SidecarImage }}
      imagePullPolicy: IfNotPresent
      {{- if gt (len .SidecarEnvs) 0 }}
      env:
      {{- range $_,$env := .SidecarEnvs }}
      - name: {{ $env.Name }}
        value: {{ $env.Value }}
      {{- end }}
      {{- end }}
      {{- if gt (len .SidecarArgs) 0 }}
      args:
      {{- range .SidecarArgs }}
      - {{ . }}
      {{- end }}
      {{- end }}
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop:
            - ALL
        privileged: false
        readOnlyRootFilesystem: true
        runAsGroup: 1337
        runAsNonRoot: true
        runAsUser: 1337
      volumeMounts:
        - name: envoyconfig
          mountPath: /etc/envoy/
  volumes:
    - name: envoyconfig
      configMap:
        defaultMode: 0655
        name: envoyconfig
        items:
          - key: envoy.yaml
            path: envoy.yaml