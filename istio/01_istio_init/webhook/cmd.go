package main

import (
	"context"
	"fishnet-inject/sugar"
	"fishnet-inject/webhooks"
	"github.com/go-logr/logr"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func main() {

	// init sugar logger
	sugar.InitLogger()
	log.SetLogger(logr.Discard()) // TODO: 我们需要实现一个controller-runtime log sink

	// 初始化WebhookServer
	sre := webhooks.NewWebhookServer()

	sugar.Info("fishnet injector webhook running on 9527")
	if err := sre.Start(context.Background()); err != nil {
		sugar.Error("fishnet injector webhook run error: ", err)
		os.Exit(1)
	}

}
