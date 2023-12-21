package webhooks

import (
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// NewWebhookServer
// 创建webhook服务
func NewWebhookServer() webhook.Server {

	sre := webhook.NewServer(webhook.Options{
		Port: 9527,
		// 将在certDir下寻找证书
		CertDir:  "D:\\treasure\\istio\\01_istio_init\\webhook",
		CertName: "tls.pem",
		KeyName:  "tls-key.pem",
	})

	injectWh := NewInjectorWebhook()
	// 自动注入webhook注册
	sre.Register("/look", injectWh)
	// 就绪探针注册(顺序很重要, 必须在webhook注册之后再进行注册, 避免空指针)
	sre.WebhookMux().HandleFunc("/readyz", readyProbe(""))

	return sre
}

func readyProbe(readyMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if readyMsg == "" {
			readyMsg = "ready\n"
		}

		w.Write([]byte(readyMsg))
	}
}
