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
/*	ip := flag.String("server", "127.0.0.1", `
	"IP address on which the API will be listening, default is 127.0.0.1"`)
	port := flag.String("port", "8080", `
	"Port on which the API will be listening, default is 8080"`)
	flag.Parse()
	server := fmt.Sprintf("%s:%s", *ip, *port)
*/

        server := server.Init(conf)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
