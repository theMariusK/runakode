package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/theMariusK/runakode/api/server"
	"github.com/theMariusK/runakode/config"
	"github.com/theMariusK/runakode/worker/worker"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	configPath := flag.String("config", "./config.yaml", "Configuration file path")
	address := flag.String("address", "127.0.0.1", "IP address on which the API server will be listening")
	port := flag.String("port", "8080", "Port on which the API server will be listening")
	flag.Parse()

	conf := config.Load(*configPath)

	if *address != "127.0.0.1" {
		conf.Address = *address
	}
	if *port != "8080" {
		conf.Port = *port
	}

	// --- start the API server ---

	srv := server.Init(conf)
	go func() {
		if err := srv.Run(); err != nil {
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

	// --- graceful shutdown ---

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range msgs {
			jobChan <- msg
		}
	}()

	<-sigChan
	slog.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv.Shutdown(shutdownCtx)

	close(jobChan)
	wg.Wait()

	slog.Info("shutdown complete")
}
