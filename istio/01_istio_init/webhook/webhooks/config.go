package webhooks

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"text/template"
)

// Config 自动注入的配置
// TODO: 后续需要添加watch功能 在k8s ConfigMap中获取
type Config struct {
	InitTemplate    *template.Template
	SidecarTemplate *template.Template
	ValueConfig     *Value
}

func (c *Config) setDefault() {

	if c.InitTemplate == nil {
		c.InitTemplate = template.Must(template.New("init-containers").Parse(initContainerTemplate))
	}

	if c.SidecarTemplate == nil {
		c.SidecarTemplate = template.Must(template.New("sidecar-containers").Parse(sidecarContainerTemplate))
	}

	if c.ValueConfig == nil {
		c.ValueConfig = newDefaultValue()
	}

}

func NewDefaultConfig() *Config {

	cfg := &Config{}
	cfg.setDefault()

	return cfg
}

type Value struct {
	// sidecar proxy env
	ProxyEnv map[string]string
	// injector extra annotations
	Annotations map[string]string
	// injector extra labels
	Labels map[string]string
	// sidecar proxy probe
	ProxyProbe *Probes
}

func newDefaultValue() *Value {

	value := &Value{}

	value.Annotations = map[string]string{}
	value.Labels = map[string]string{}
	value.ProxyEnv = map[string]string{}

	probes := &Probes{}
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
	value.ProxyProbe = probes

	return value
}

type Probes struct {
	StartupProbe   *corev1.Probe
	ReadinessProbe *corev1.Probe
}

type TemplateData struct {
}
