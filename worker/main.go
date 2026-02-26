package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/theMariusK/runakode/config"
	"github.com/theMariusK/runakode/worker/worker"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	configPath := flag.String("config", "./config.yaml", "Configuration file path")
	flag.Parse()

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
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	msgs, err := ch.Consume(
		conf.RabbitMQ.Queue,
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	jobChan := make(chan amqp.Delivery, conf.MaxWorkers*2)

	var wg sync.WaitGroup
	for i := 0; i < conf.MaxWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker.Worker(id, conn, jobChan, conf)
		}(i)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range msgs {
			jobChan <- msg
		}
	}()

	<-sigChan
	slog.Info("shutdown signal received")

	close(jobChan)
	wg.Wait()

	slog.Info("shutdown complete")
}
