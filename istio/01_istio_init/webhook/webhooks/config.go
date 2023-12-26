package webhooks

import "text/template"

// Config 自动注入的配置
// TODO: 后续需要添加watch功能
// TODO: 我们考虑从以下两个途径获取配置信息
// +在k8s ConfigMap中获取
// +在webhook启动时读取文件
type Config struct {
	InitTemplate    *template.Template
	SidecarTemplate *template.Template
}

func (c *Config) setDefault() {

	if c.InitTemplate == nil {
		c.InitTemplate = template.Must(template.New("init-containers").Parse(initContainerTemplate))
	}

	if c.SidecarTemplate == nil {
		c.SidecarTemplate = template.Must(template.New("sidecar-containers").Parse(sidecarContainerTemplate))
	}

}

func NewDefaultConfig() *Config {

	cfg := &Config{}
	cfg.setDefault()

	return cfg
}

type TemplateData struct {
}

const initContainerTemplate = `
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
        - *
        - -x
        - ""
        - -b
        - *
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
  volume:
	- name: envoyconfig
      configMap:
        defaultMode: 0655
        name: envoyconfig
		items:
		  - key: envoy.yaml
			path: envoy.yaml
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
`
