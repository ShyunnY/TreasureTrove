apiVersion: v1
kind: Pod
spec:
  initContainers:
    - image: {{ .InitContainerImage }}
      name: {{ .InitContainerName }}
      imagePullPolicy: IfNotPresent
      {{- if gt (len .InitContainerArgs) 0 }}
      args:
      {{- range .InitContainerArgs }}
      - {{ . }}
      {{- end }}
      {{- end }}
      resources:
        limits:
          cpu: "2"
          memory: 1Gi
        requests:
          cpu: 100m
          memory: 128Mi
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          add:
            - NET_ADMIN
            - NET_RAW
          drop:
            - ALL
        privileged: false
        readOnlyRootFilesystem: false
        runAsGroup: 0
        runAsNonRoot: false
        runAsUser: 0