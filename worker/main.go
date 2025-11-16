package main

import (
	"github.com/theMariusK/runakode/api/config"
	"flag"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"encoding/json"
	"github.com/theMariusK/runakode/worker/runner"
)

func main() {
	// initial configuration
	configPath := flag.String("config", "../config.yaml", `
	"Configuration file path, default is ../config.yaml"`)
	conf := config.Load(*configPath)

	conn, err := amqp.Dial(conf.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("Can't connect to RabbitMQ (%s)!\n", conf.RabbitMQ.URL)
	}
	defer conn.Close()

	ch, err := conn.Channel()
        if err != nil {
                log.Fatal(err.Error())
        }
	defer ch.Close()

	_, err = ch.QueueDeclare(
                conf.RabbitMQ.Queue,
                true, // durable
                false, // autoDelete
                false, // exclusive
                false, // noWait
                nil, // args
        )
	if err != nil {
		log.Fatal(err.Error())
	}

	msgs, err := ch.Consume(
		conf.RabbitMQ.Queue, // queue
		"", // consumer
		true, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil, // args
	)

	log.Println("Listening for messages...")

	var forever chan struct{}

	go func() {
		for msg := range msgs {
			log.Printf("Got a message: %s\n", msg.Body)

			var request *runner.RunRequest
			err := json.Unmarshal([]byte(msg.Body), &request)
			if err != nil {
				log.Println(err.Error())
                                return
			}

			response := runner.RunSandbox(request)

			ch.Publish(
				"",
				msg.ReplyTo,
				false,
				false,
				amqp.Publishing{
					ContentType: "application/json",
					CorrelationId: msg.CorrelationId,
					Body: []byte(response),
				},
			)
		}
	}()

	<-forever
}
