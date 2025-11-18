package handlers

import (
	"fmt"
	"context"
	"net/http"
	"log"
	"encoding/json"
	"github.com/theMariusK/runakode/config"
	"slices"
	"time"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/google/uuid"
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

func SendAndWait(mq *amqp.Channel, queue string, job []byte, timeout int) ([]byte, error) {
	corrID := uuid.New().String()

	reply, err := mq.QueueDeclare(
		"",
		false, // durable
		true, // autoDelete
		true, // exclusive
		false, // noWait
		nil, //args
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

        ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout) * time.Second)
	defer cancel()

	err = mq.PublishWithContext(
		ctx,
		"",
		queue,
		false,
		false,
		amqp.Publishing {
			ContentType: "application/json",
			Body: job,
			ReplyTo: reply.Name,
			CorrelationId: corrID,
		})
	if err != nil {
		log.Printf("Message has not been sent!\n%v", err.Error())
	}

	for msg := range msgs {
		if msg.CorrelationId == corrID {
			return msg.Body, nil
		}
	}

	return nil, fmt.Errorf("No matching response found!")
}

func Api(conf *config.Config, mq *amqp.Channel) (http.HandlerFunc) {
	return func(w http.ResponseWriter, r *http.Request) {
	        log.Printf("Got a %s request from %s\n", r.Method, r.RemoteAddr)
	        if r.Method != http.MethodPost {
		        http.Error(w, "Wrong method!", http.StatusMethodNotAllowed)
		        return
	        }

	        var request RunRequest
	        err := json.NewDecoder(r.Body).Decode(&request)
	        if err != nil {
	 	        log.Println(err.Error())
		        http.Error(w, "Cant parse the request!", http.StatusBadRequest)
		        return
	        }

		if ! slices.Contains(conf.SupportedLanguages, request.Language) {
			http.Error(w, "Unsupported language!", http.StatusBadRequest)
			return
		}

		job, err := json.Marshal(request)
		response, err := SendAndWait(mq, conf.RabbitMQ.Queue, job, conf.ApiTimeout)
		if err != nil {
			log.Println(err.Error())
		}
		fmt.Fprint(w, string(response))
        }
}
