package webhooks

var IgonreNamespace = []string{
	"kube-system",
	"kube-public",
	"kube-node-lease",
}

// injector common annotation
var (
	sidecarInjectAnnotation       = "sidecar.fishnet.io/inject"
	sidecarIgnoreInjectAnnotation = "sidecar.fishnet.io/ignore"
	sidecarOverwriteAnnotation    = "sidecar.fishnet.io/overwrite.probe"
	customTemplateAnnotation      = "inject.fishnet.io/template"
)

const (
	ProxyContainerName   = "envoyproxy"
	InjectorConfigMapKey = "fishnet-injector-config"
)

const configNamespace = "mesh"

const (
	moveLast = iota
	moveFirst
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
      name: fishnet-init
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
const proxyConfigTemplate = `
static_resources:
  listeners:
    - name: listener_outbound
      traffic_direction: OUTBOUND
      use_original_dst: true
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 15001
      filter_chains:
        - name: virtualOutbound-catchall-tcp
          filters:
            - name: envoy.filters.network.tcp_proxy
              typed_config:
                '@type': type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
                cluster: PassThroughCluster
                stat_prefix: PassThroughCluster
    - name: listener_inbound
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 15006
      traffic_direction: INBOUND
      listener_filters_timeout: 0s
      listener_filters:
        - name: envoy.filters.listener.original_dst
          typed_config:
            '@type': type.googleapis.com/envoy.extensions.filters.listener.original_dst.v3.OriginalDst
        - name: envoy.filters.listener.tls_inspector
          typed_config:
            '@type': type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector
          filter_disabled:
            destination_port_range:
              start: 15006
              end: 15007
        - name: envoy.filters.listener.http_inspector
          typed_config:
            '@type': type.googleapis.com/envoy.extensions.filters.listener.http_inspector.v3.HttpInspector
          filter_disabled:
            or_match:
              rules:
                - destination_port_range:
                    start: 80
                    end: 81
                - destination_port_range:
                    start: 15006
                    end: 15007
      filter_chains:
        - name: inbound|80|listener
          filter_chain_match:
            destination_port: 80
            transport_protocol: raw_buffer
          filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                upgrade_configs:
                  - upgrade_type: websocket
                codec_type: AUTO
                use_remote_address: false
                normalize_path: true
                path_with_escaped_slashes_action: KEEP_UNCHANGED
                access_log:
                  - name: envoy.access_loggers.stdout
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
                route_config:
                  name: inbound||80
                  validate_clusters: false
                  virtual_hosts:
                    - name: inbound|http|80
                      domains: [ "*" ]
                      routes:
                        - match:
                            prefix: /
                          route:
                            cluster: inbound|80|
                http_filters:
                  - name: envoy.filters.http.router
                    typed_config:
                      '@type': type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
  clusters:
    - name: inbound|80|
      common_lb_config: { }
      connect_timeout: 1s
      type: ORIGINAL_DST
      lbPolicy: CLUSTER_PROVIDED
      upstream_bind_config:
        source_address:
          address: 127.0.0.6
          portValue: 0
    - name: PassthroughCluster
      connect_timeout: 1s
      type: ORIGINAL_DST
      lbPolicy: CLUSTER_PROVIDED
      typed_extension_protocol_options:
        envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
          '@type': type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
          common_http_protocol_options:
            idle_timeout: 300s
          use_downstream_protocol_config:
            http_protocol_options: { }
            http2_protocol_options: { }
`
