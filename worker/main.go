package main

import (
	"log"
	"flag"
	"github.com/theMariusK/runakode/config"
	"github.com/theMariusK/runakode/worker/worker"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// --- initial configuration ---
	configPath := flag.String("config", "./config.yaml", "Configuration file path")
	conf := config.Load(*configPath)

	// --- start the Worker ---

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
                conf.RabbitMQ.Queue, // queue
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
	if err != nil {
		log.Fatal(err.Error())
	}

	jobChan := make(chan amqp.Delivery, conf.MaxWorkers * 2)

	for i := 0; i < conf.MaxWorkers; i++ {
		go worker.Worker(i, conn, jobChan, conf)
	}

	for msg := range msgs {
		jobChan <- msg
	}
}
