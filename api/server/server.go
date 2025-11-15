package server

import (
	"net/http"
	"github.com/theMariusK/runakode/api/config"
	"github.com/theMariusK/runakode/api/handlers"
	"fmt"
	"log"
)

type APIServer struct {
	conf *config.Config
}

func Init(conf *config.Config) (*APIServer) {
	return &APIServer{conf: conf}
}

func (s *APIServer) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", handlers.Api(s.conf))

	server := fmt.Sprintf("%s:%s", s.conf.Address, s.conf.Port)
	log.Printf("Server is listening on %s!\n", server)
	return http.ListenAndServe(server, mux)
}
