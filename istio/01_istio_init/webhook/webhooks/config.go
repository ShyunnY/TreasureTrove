package webhooks

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
	"log"
)

// Config 自动注入的配置
type Config struct {

	// ValueConfig用于控制injector的行为.
	// 例如: 控制被注入Pod的labels,annotations, sidecarEnv, CustomTemplate等
	ValueConfig *Value `json:"valueConfig" yaml:"valueConfig"`

	// TODO: 目前我们使用字符串代替, 将来会使用go-control-plane的Envoy proto
	// TODO: base config我认为将直接存放在envoy中
	ProxyConfig any `json:"proxyConfig" yaml:"proxyConfig"`

	td *templateData
}

func NewDefaultConfig() *Config {

	cfg := &Config{}
	cfg.setDefault()

	return cfg
}

func (c *Config) setDefault() {

	// 不设置proxyConfig, 或者将ProxyConfig显式设置为空
	if c.ProxyConfig == nil {
		// TODO: 将来会使用xds时候, 我们会进行自动注入配置
	}

	if c.td == nil {
		c.td = NewTemplateData()
	}

	c.ValueConfig = newDefaultValue(c.ValueConfig)

}

func NewTemplateData() *templateData {

	td := &templateData{
		SidecarName:        "fishnet-proxy",
		SidecarImage:       "envoyproxy/envoy-alpine:v1.21.0",
		SidecarEnvs:        []corev1.EnvVar{},
		SidecarArgs:        []string{},
		InitContainerName:  "fishnet-init",
		InitContainerImage: "docker.io/istio/proxyv2:1.20.1",
		InitContainerArgs:  initContainerArgs,
	}

	return td
}

func (c *Config) GetTemplateData() *templateData {
	return c.td
}

func unmarshalConfig(data []byte) (*Config, error) {
	var injectConfig Config
	if err := yaml.Unmarshal(data, &injectConfig); err != nil {
		log.Println("unmarshal injector config error: ", err)

		return nil, fmt.Errorf("unmarshal injector config error: %v", err)
	}

	// robustness checking
	injectConfig.setDefault()

	return &injectConfig, nil
}

type Value struct {
	// sidecar proxy env
	SidecarEnv map[string]string `json:"proxyEnv,omitempty" yaml:"proxyEnv"`
	// injector extra annotations
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations"`
	// injector extra labels
	Labels map[string]string `json:"labels,omitempty" yaml:"labels"`
	// sidecar proxy probe
	ProxyProbe *Probes `json:"proxyProbe,omitempty" yaml:"proxyProbe"`
	// explicitly set to false turns off probe injection, default turn on probe injection
	InjectProbe *bool `json:"injectProbe" yaml:"injectProbe"`

	CustomTemplate string `json:"customTemplate" yaml:"customTemplate"`

	// whether to wait for the proxy to start before starting the application;
	// defaults to false
	AfterProxyStart bool `yaml:"afterProxyStart" yaml:"afterProxyStart"`
}

func newDefaultValue(origin *Value) *Value {

	if origin == nil {
		origin = &Value{}
	}

	if origin.Labels == nil {
		origin.Labels = map[string]string{}
	}

	if origin.Annotations == nil {
		origin.Annotations = map[string]string{}
	}

	if origin.SidecarEnv == nil {
		origin.SidecarEnv = map[string]string{}
	}
	// 如果没有明确将injectProbe设置为false, 我们都注入探针
	if origin.InjectProbe != nil && *origin.InjectProbe == false {
		return origin
	}

	probes := Probes{}
	// TODO: 探针port需要与Dynamic Config, InitContainer的iptables进行同步
	probes.ReadinessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/healthz/ready",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 15021,
				},
				Scheme: corev1.URISchemeHTTP,
			},
		},
		// 3s超时
		TimeoutSeconds: 3,
		// 执行周期为1s
		PeriodSeconds: 15,
		// 执行成功1次, 我们认为此探针成功
		SuccessThreshold: 1,
		// 执行失败4次, 我们认为此探针失败
		FailureThreshold: 4,
	}
	probes.StartupProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/healthz/ready",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 15021,
				},
				Scheme: corev1.URISchemeHTTP,
			},
		},
		TimeoutSeconds:   3,
		PeriodSeconds:    1,
		SuccessThreshold: 1,
		FailureThreshold: 600,
	}

	if origin.ProxyProbe == nil {
		origin.ProxyProbe = &probes
	} else {

		if origin.ProxyProbe.ReadinessProbe == nil {
			origin.ProxyProbe.ReadinessProbe = probes.ReadinessProbe
		}

		if origin.ProxyProbe.StartupProbe == nil {
			origin.ProxyProbe.StartupProbe = probes.StartupProbe
		}

	}

	return origin
}

type Probes struct {
	StartupProbe   *corev1.Probe `json:"startupProbe,omitempty" yaml:"startupProbe"`
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty" yaml:"readinessProbe"`
}

// TemplateData
// 将内联到Config中
type templateData struct {
	// sidecar config
	SidecarName  string
	SidecarImage string
	SidecarEnvs  []corev1.EnvVar
	SidecarArgs  []string

	InitContainerName  string
	InitContainerImage string
	InitContainerArgs  []string
}
