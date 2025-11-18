package main

import (
	"log"
	"flag"
	"github.com/theMariusK/runakode/config"
	"github.com/theMariusK/runakode/api/server"
	"github.com/theMariusK/runakode/worker/worker"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// --- initial configuration ---
	configPath := flag.String("config", "./config.yaml", "Configuration file path")
	conf := config.Load(*configPath)

	// --- ability to override configuration ---
	default_values := map[string]string{
		"address": "127.0.0.1",
		"port": "8080",
	}

	address := flag.String("address", default_values["address"], "IP address on which the API server will be listening")
	port := flag.String("port", default_values["port"], "Port on which the API server will be listening")
	flag.Parse()

	if *address != default_values["address"] {
		conf.Address = *address
	}

	if *port != default_values["port"] {
		conf.Port = *port
	}

	// --- start the API server ---

	go func() {
		server := server.Init(conf)
		if err := server.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

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
