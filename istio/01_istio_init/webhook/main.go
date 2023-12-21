package main

import (
	"context"
	"log"
	"webhook/webhooks"
)

// This is an example webhook
func main() {

	sre := webhooks.NewWebhookServer()

	log.Println("fishnet injector webhook running on 9527")
	if err := sre.Start(context.Background()); err != nil {

		log.Fatalln("fishnet injector webhook start error: ", err.Error())
	}

}
