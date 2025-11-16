package main

import (
	"github.com/theMariusK/runakode/worker/config"
	"flag"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
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
	if err != nil {
		log.Fatal(err.Error())
	}

	jobChan := make(chan amqp.Delivery, conf.MaxJobs * 2)

	for i := 0; i < conf.MaxJobs; i++ {
		go runner.Worker(i, conn, jobChan)
	}

	for msg := range msgs {
		jobChan <- msg
	}
}
