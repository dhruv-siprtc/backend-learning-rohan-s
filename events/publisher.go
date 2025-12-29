package events

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"golang-postgre/config"
)

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

func PublishUserCreated(id uint, name, email string) {
	publish(UserEvent{
		Event:     "USER_CREATED",
		Version:   "1.0",
		Timestamp: time.Now(),
		Data: UserData{
			UserID: id,
			Name:   name,
			Email:  email,
		},
	}, "user.created")
}

func PublishUserUpdated(id uint, name, email string) {
	publish(UserEvent{
		Event:     "USER_UPDATED",
		Version:   "1.0",
		Timestamp: time.Now(),
		Data: UserData{
			UserID: id,
			Name:   name,
			Email:  email,
		},
	}, "user.updated")
}
