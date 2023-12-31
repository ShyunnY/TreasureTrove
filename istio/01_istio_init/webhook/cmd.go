package main

import (
	"context"
	"fishnet-inject/webhooks"
	"log"
)

func main() {

	// 初始化WebhookServer
	sre := webhooks.NewWebhookServer()

	log.Println("fishnet injector webhook running on 9527")
	if err := sre.Start(context.Background()); err != nil {

		log.Fatalln("fishnet injector webhook start error: ", err.Error())
	}

}
