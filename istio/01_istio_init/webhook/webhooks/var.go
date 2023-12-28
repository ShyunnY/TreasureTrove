package webhooks

var IgonreNamespace = []string{
	"kube-system",
	"kube-public",
	"kube-node-lease",
}

var (
	sidecarInjectAnnotation       = "sidecar.fishnet.io/inject"
	sidecarIgnoreInjectAnnotation = "sidecar.fishnet.io/ignore"
	sidecarOverwriteAnnotation    = "sidecar.fishnet.io/overwrite.probe"
)

const (
	ProxyContainerName   = "envoyproxy"
	InjectorConfigMapKey = "fishnet-injector-config"
)

const initContainerTemplate = `
apiVersion: v1
kind: Pod
spec:
  initContainers:
    - args:
        - istio-iptables
        - -p
        - "15001"
        - -z
        - "15006"
        - -u
        - "1337"
        - -m
        - REDIRECT
        - -i
        - '*'
        - -x
        - ""
        - -b
        - '*'
        - -d
        - 15090,15021,15020
        - --log_output_level=default:info
      image: docker.io/istio/proxyv2:1.20.1
      imagePullPolicy: IfNotPresent
      name: istio-init
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
`

const sidecarContainerTemplate = `
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: envoyproxy
      image: envoyproxy/envoy-alpine:v1.21.0
      imagePullPolicy: IfNotPresent
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
`
