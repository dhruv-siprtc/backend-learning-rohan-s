package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang-postgre/config"
	"golang-postgre/events"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	config.ConnectRabbitMQ()
	defer func() {
		if config.RabbitChannel != nil {
			config.RabbitChannel.Close()
			log.Println("RabbitMQ channel closed")
		}
		if config.RabbitConn != nil {
			config.RabbitConn.Close()
			log.Println(" RabbitMQ connection closed")
		}
	}()

	createdQueue := os.Getenv("RABBITMQ_CREATED_QUEUE")
	updatedQueue := os.Getenv("RABBITMQ_UPDATED_QUEUE")

	if createdQueue == "" {
		log.Fatal("RABBITMQ_CREATED_QUEUE not set in environment")
	}
	if updatedQueue == "" {
		log.Fatal(" RABBITMQ_UPDATED_QUEUE not set in environment")
	}

	createdMsgs, err := config.RabbitChannel.Consume(
		createdQueue,
		"created-consumer",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to consume created queue (%s): %v", createdQueue, err)
	}

	updatedMsgs, err := config.RabbitChannel.Consume(
		updatedQueue,
		"updated-consumer",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to consume updated queue (%s): %v", updatedQueue, err)
	}

	log.Printf("Consumer started. Listening to queues: %s, %s\n", createdQueue, updatedQueue)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			select {
			case msg, ok := <-createdMsgs:
				if !ok {
					log.Println(" Created queue channel closed")
					return
				}

				var event events.UserEvent
				if err := json.Unmarshal(msg.Body, &event); err != nil {
					log.Println("Invalid message in created queue:", err)
					msg.Nack(false, false)
					continue
				}

				log.Printf("[USER_CREATED] Welcome email sent to %s (UserID: %d)\n", event.Data.Email, event.Data.UserID)
				msg.Ack(false)

			case <-ctx.Done():
				log.Println(" Stopping created queue consumer...")
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case msg, ok := <-updatedMsgs:
				if !ok {
					log.Println(" Updated queue channel closed")
					return
				}

				var event events.UserEvent
				if err := json.Unmarshal(msg.Body, &event); err != nil {
					log.Println(" Invalid message in updated queue:", err)
					msg.Nack(false, false)
					continue
				}

				log.Printf("[USER_UPDATED] User %d (%s) profile updated\n", event.Data.UserID, event.Data.Email)
				msg.Ack(false)

			case <-ctx.Done():
				log.Println("Stopping updated queue consumer...")
				return
			}
		}
	}()

	<-sigChan
	log.Println("Shutdown signal received, closing connections...")
	cancel()
}
