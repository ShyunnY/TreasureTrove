package webhooks

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
	"log"
	"text/template"
)

// Config 自动注入的配置
type Config struct {
	InitTemplate    *template.Template `json:"initTemplate" yaml:"initTemplate"`
	SidecarTemplate *template.Template `json:"sidecarTemplate" yaml:"sidecarTemplate"`
	ValueConfig     *Value             `json:"valueConfig" yaml:"valueConfig"`
}

func (c *Config) setDefault() {

	if c.InitTemplate == nil {
		c.InitTemplate = template.Must(template.New("init-containers").Parse(initContainerTemplate))
	}

	if c.SidecarTemplate == nil {
		c.SidecarTemplate = template.Must(template.New("sidecar-containers").Parse(sidecarContainerTemplate))
	}

	newDefaultValue(c.ValueConfig)

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

func NewDefaultConfig() *Config {

	cfg := &Config{}
	cfg.setDefault()

	return cfg
}

type Value struct {
	// sidecar proxy env
	ProxyEnv map[string]string `json:"proxyEnv,omitempty" yaml:"proxyEnv"`
	// injector extra annotations
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations"`
	// injector extra labels
	Labels map[string]string `json:"labels,omitempty" yaml:"labels"`
	// sidecar proxy probe
	ProxyProbe *Probes `json:"proxyProbe,omitempty" yaml:"proxyProbe"`

	injectProbe *bool
}

func newDefaultValue(origin *Value) *Value {

	if origin.Labels == nil {
		origin.Labels = map[string]string{}
	}

	if origin.Annotations == nil {
		origin.Annotations = map[string]string{}
	}

	if origin.ProxyEnv == nil {
		origin.ProxyEnv = map[string]string{}
	}

	// 如果没有明确将injectProbe设置为false, 我们都注入探针
	if origin.injectProbe != nil && *origin.injectProbe == false {
		return origin
	}

	probes := Probes{}
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

type TemplateData struct {
}
