package config

import (
	"errors"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishUserCreated(body []byte) error {
	if RabbitChannel == nil {
		return errors.New("rabbitmq channel not initialized")
	}

	return RabbitChannel.Publish(
		os.Getenv("RABBITMQ_EXCHANGE"),
		"user.created",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func PublishUserUpdated(body []byte) error {
	if RabbitChannel == nil {
		return errors.New("rabbitmq channel not initialized")
	}

	return RabbitChannel.Publish(
		os.Getenv("RABBITMQ_EXCHANGE"),
		"user.updated",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}
