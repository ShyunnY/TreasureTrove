package webhooks

import _ "embed"

var IgonreNamespace = []string{
	"kube-system",
	"kube-public",
	"kube-node-lease",
}

// injector common annotation
var (
	sidecarInjectAnnotation       = "sidecar.fishnet.io/inject"
	sidecarIgnoreInjectAnnotation = "sidecar.fishnet.io/ignore"
	sidecarInjectStatus           = "sidecar.fishnet.io/status"
	sidecarOverwriteAnnotation    = "sidecar.fishnet.io/overwrite.probe"
	customTemplateAnnotation      = "inject.fishnet.io/template"
)

// injector common name
const (
	ProxyContainerName   = "envoyproxy"
	InjectorConfigMapKey = "fishnet-injector-config"
	configNamespace      = "mesh"
)

// reorder action
const (
	moveLast = iota
	moveFirst
)

// initContainer args
var initContainerArgs = []string{
	"istio-iptables",
	"-p",
	`"15001"`,
	"-z",
	`"15006"`,
	"-u",
	`"1337"`,
	"-m",
	"REDIRECT",
	"-i",
	`'*'`,
	`-x`,
	`""`,
	"-b",
	`'*'`,
	"-d",
	"15090,15021,15020",
	"--log_output_level=default:info",
}
