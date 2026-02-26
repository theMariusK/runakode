package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/theMariusK/runakode/config"
)

type RunRequest struct {
	Language   string `json:"language"`
	SourceCode string `json:"source_code"`
}

func SendAndWait(mq *amqp.Channel, queue string, job []byte, timeout int) ([]byte, error) {
	corrID := uuid.New().String()

	reply, err := mq.QueueDeclare(
		"",
		false, // durable
		true,  // autoDelete
		true,  // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	msgs, err := mq.Consume(
		reply.Name,
		"",
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	err = mq.PublishWithContext(
		ctx,
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          job,
			ReplyTo:       reply.Name,
			CorrelationId: corrID,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return nil, fmt.Errorf("reply channel closed unexpectedly")
			}
			if msg.CorrelationId == corrID {
				return msg.Body, nil
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for response")
		}
	}
}

func Api(conf *config.Config, mq *amqp.Channel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("received request", "method", r.Method, "remote", r.RemoteAddr)

		if r.Method != http.MethodPost {
			http.Error(w, "Wrong method!", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

		var request RunRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			slog.Error("failed to parse request", "error", err)
			http.Error(w, "Cant parse the request!", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(request.SourceCode) == "" {
			http.Error(w, "source_code must not be empty!", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(request.Language) == "" {
			http.Error(w, "language must not be empty!", http.StatusBadRequest)
			return
		}

		if !slices.Contains(conf.SupportedLanguages, request.Language) {
			http.Error(w, "Unsupported language!", http.StatusBadRequest)
			return
		}

		job, err := json.Marshal(request)
		if err != nil {
			slog.Error("failed to marshal job", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response, err := SendAndWait(mq, conf.RabbitMQ.Queue, job, conf.ApiTimeout)
		if err != nil {
			slog.Error("job execution failed", "error", err)
			http.Error(w, "Gateway timeout", http.StatusGatewayTimeout)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}
