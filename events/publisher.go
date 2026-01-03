package events

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"golang-postgre/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Generic publish function (kept for reference or future use)
func publish(event UserEvent, routingKey string) {
	body, _ := json.Marshal(event)

	err := config.RabbitChannel.PublishWithContext(
		context.Background(),
		os.Getenv("RABBITMQ_EXCHANGE"),
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		log.Println("❌ Event publish failed:", err)
	}
}
