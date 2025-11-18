package main

import (
	"log"
	"flag"
	"github.com/theMariusK/runakode/config"
	"github.com/theMariusK/runakode/api/server"
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

	server := server.Init(conf)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
