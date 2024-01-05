package webhooks

import (
	"fishnet-inject/sugar"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type WebhookServerConfig struct {
	Port     int
	CertDir  string
	CertName string
	KeyName  string
}

func (wsc *WebhookServerConfig) Default() {

	if wsc.Port == 0 {
		wsc.Port = 9527
	}

	if wsc.CertDir == "" {
		// TODO: Warnings! mock data use on the test machine only
		wsc.CertDir = "D:\\treasure\\istio\\01_istio_init\\webhook"
	}

	if wsc.CertName == "" {
		// TODO: Warnings! mock data use on the test machine only
		wsc.CertName = "tls.pem"
	}

	if wsc.KeyName == "" {
		// TODO: Warnings! mock data use on the test machine only
		wsc.KeyName = "tls-key.pem"
	}

}

// NewWebhookServer
// 创建webhook服务
func NewWebhookServer(conf WebhookServerConfig) webhook.Server {

	conf.Default()

	sre := webhook.NewServer(webhook.Options{
		Port:     conf.Port,
		CertDir:  conf.CertDir,
		CertName: conf.CertName,
		KeyName:  conf.KeyName,
	})

	injectWh, err := NewInjectorWebhook()
	if err != nil {
		sugar.Error("create inject webhook error: ", err)
		os.Exit(1)
	}

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
