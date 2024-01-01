package main

import (
	"context"
	"fishnet-inject/sugar"
	"fishnet-inject/webhooks"
	"os"
)

func main() {

	// 初始化WebhookServer
	sre := webhooks.NewWebhookServer()

	sugar.Info("fishnet injector webhook running on 9527")
	if err := sre.Start(context.Background()); err != nil {
		sugar.Error("fishnet injector webhook run error: ", err)
		os.Exit(1)
	}

}
