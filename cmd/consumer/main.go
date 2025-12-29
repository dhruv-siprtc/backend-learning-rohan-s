package main

import (
	"encoding/json"
	"log"

	"golang-postgre/config"
	"golang-postgre/events"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	config.ConnectRabbitMQ()

	msgs, _ := config.RabbitChannel.Consume(
		"",
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	for msg := range msgs {
		var event events.UserEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			msg.Nack(false, false)
			continue
		}

		switch event.Event {
		case "USER_CREATED":
			log.Printf("[USER_CREATED] Welcome email sent to %s\n", event.Data.Email)

		case "USER_UPDATED":
			log.Printf("[USER_UPDATED] User %d profile updated\n", event.Data.UserID)
		}

		msg.Ack(false)
	}
}
