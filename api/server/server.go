package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/theMariusK/runakode/api/handlers"
	"github.com/theMariusK/runakode/api/middleware"
	"github.com/theMariusK/runakode/config"
)

type APIServer struct {
	conf       *config.Config
	mqConn     *amqp.Connection
	mqCh       *amqp.Channel
	httpServer *http.Server
}

func Init(conf *config.Config) *APIServer {
	conn, err := amqp.Dial(conf.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("Can't connect to RabbitMQ (%s)!\n", conf.RabbitMQ.URL)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err.Error())
	}

	return &APIServer{
		conf:   conf,
		mqConn: conn,
		mqCh:   ch,
	}
}

func (s *APIServer) Run() error {
	_, err := s.mqCh.QueueDeclare(
		s.conf.RabbitMQ.Queue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		log.Fatal(err.Error())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api", handlers.Api(s.conf, s.mqCh))
	mux.HandleFunc("/health", s.healthHandler)

	rl := middleware.NewRateLimiter(10, 20)
	handler := middleware.RequestID(middleware.CORS(rl.Middleware(mux)))

	addr := fmt.Sprintf("%s:%s", s.conf.Address, s.conf.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	slog.Info("server listening", "address", addr)
	return s.httpServer.ListenAndServe()
}

func (s *APIServer) Shutdown(ctx context.Context) error {
	slog.Info("shutting down API server")

	var httpErr error
	if s.httpServer != nil {
		httpErr = s.httpServer.Shutdown(ctx)
	}

	if s.mqCh != nil {
		s.mqCh.Close()
	}
	if s.mqConn != nil {
		s.mqConn.Close()
	}

	return httpErr
}

func (s *APIServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Wrong method!", http.StatusMethodNotAllowed)
		return
	}

	status := "ok"
	if s.mqConn.IsClosed() {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status})
}
