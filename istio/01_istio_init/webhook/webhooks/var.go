package webhooks

var IgonreNamespace = []string{
	"kube-system",
	"kube-public",
	"kube-node-lease",
}

var (
	sidecarInjectAnnotation       = "sidecar.fishnet.io/inject"
	sideacrIgnoreInjectAnnotation = "sidecar.fishnet.io/ignore"
)
