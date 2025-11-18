package server

import (
	"net/http"
	"github.com/theMariusK/runakode/config"
	"github.com/theMariusK/runakode/api/handlers"
	"fmt"
	"log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type APIServer struct {
	conf *config.Config
	mqConn *amqp.Connection
	mqCh *amqp.Channel
}

func Init(conf *config.Config) (*APIServer) {
	conn, err := amqp.Dial(conf.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("Can't connect to RabbitMQ (%s)!\n", conf.RabbitMQ.URL)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err.Error())
	}

	return &APIServer{
		conf: conf,
		mqConn: conn,
		mqCh: ch,
	}
}

func (s *APIServer) Run() error {
	_, err := s.mqCh.QueueDeclare(
                s.conf.RabbitMQ.Queue,
                true, // durable
                false, // autoDelete
                false, // exclusive
                false, // noWait
                nil, // args
        )

        if err != nil {
                log.Fatal(err.Error())
        }

	mux := http.NewServeMux()
	mux.HandleFunc("/api", handlers.Api(s.conf, s.mqCh))

	server := fmt.Sprintf("%s:%s", s.conf.Address, s.conf.Port)
	log.Printf("Server is listening on %s!\n", server)
	return http.ListenAndServe(server, mux)
}
