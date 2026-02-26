package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/theMariusK/runakode/api/server"
	"github.com/theMariusK/runakode/config"
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

	srv := server.Init(conf)

	go func() {
		if err := srv.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("shutdown complete")
}
