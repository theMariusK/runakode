package main

import (
	"fmt"
	"net/http"
	"log"
	"flag"
	"encoding/json"
	"github.com/theMariusK/runakode/api/config"
)

type RunRequest struct {
	Language string `json:"language"`
	SourceCode string `json:"source_code"`
}

type RunResponse struct {
	id int `json:"id"`
	result string `json:"result"`
	exitCode int `json:"exit_code"`
}

func api(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got a %s request from %s\n", r.Method, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "Wrong Method!", http.StatusMethodNotAllowed)
		return
	}

	var request RunRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad Request!", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, request.SourceCode)
}

func main() {
	conf := config.Load("config.yaml")
	fmt.Println(conf)

	// ability to override configuration
	ip := flag.String("server", "127.0.0.1", `
	"IP address on which the API will be listening, default is 127.0.0.1"`)
	port := flag.String("port", "8080", `
	"Port on which the API will be listening, default is 8080"`)
	flag.Parse()
	server := fmt.Sprintf("%s:%s", *ip, *port)

	log.Printf("Starting the API server on %s!\n", server)
	http.HandleFunc("/api", api)
	http.ListenAndServe(server, nil)
}
