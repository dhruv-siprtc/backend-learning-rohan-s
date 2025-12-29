package config

import (
	"fmt"
	"log"
	"os"
)

var RabbitConn *amqp.Connection
var RabbitChannel *amqp.Channel

func ConnectRabbitMQ() {
	url := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		os.Getenv("RABBITMQ_USER"),
		os.Getenv("RABBITMQ_PASSWORD"),
		os.Getenv("RABBITMQ_HOST"),
		os.Getenv("RABBITMQ_PORT"),
	)

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatal("❌ RabbitMQ connection failed:", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("❌ RabbitMQ channel failed:", err)
	}

	exchange := os.Getenv("RABBITMQ_EXCHANGE")

	// Exchange
	ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)

	// DLX
	dlx := os.Getenv("RABBITMQ_DLX")
	ch.ExchangeDeclare(dlx, "fanout", true, false, false, false, nil)

	// Queues
	createdQueue := os.Getenv("RABBITMQ_CREATED_QUEUE")
	updatedQueue := os.Getenv("RABBITMQ_UPDATED_QUEUE")

	args := amqp.Table{
		"x-dead-letter-exchange": dlx,
	}

	ch.QueueDeclare(createdQueue, true, false, false, false, args)
	ch.QueueDeclare(updatedQueue, true, false, false, false, args)

	ch.QueueBind(createdQueue, "user.created", exchange, false, nil)
	ch.QueueBind(updatedQueue, "user.updated", exchange, false, nil)

	RabbitConn = conn
	RabbitChannel = ch

	log.Println("✅ RabbitMQ connected")
}
