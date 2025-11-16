package main

import (
	"log"
	"flag"
	"github.com/theMariusK/runakode/api/config"
	"github.com/theMariusK/runakode/api/server"
)

func main() {
	// initial configuration
	configPath := flag.String("config", "../config.yaml", `
	"Configuration file path, default is ../config.yaml"`)
	conf := config.Load(*configPath)

	// ability to override configuration
	default_values := map[string]string{
		"address": "127.0.0.1",
		"port": "8080",
	}

	address := flag.String("address", default_values["address"], `
	"IP address on which the API will be listening, default is 127.0.0.1"`)
	port := flag.String("port", default_values["port"], `
	"Port on which the API will be listening, default is 8080"`)
	flag.Parse()

	if *address != default_values["address"] {
		conf.Address = *address
	}

	if *port != default_values["port"] {
		conf.Port = *port
	}

        server := server.Init(conf)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
