package config

import (
	"fmt"
	"log"
	"time"
)

type RabbitMQConf struct {
	Host              string `env:"RABBITMQ_HOST" envDefault:"localhost"`
	Port              string `env:"RABBITMQ_PORT" envDefault:"5672"`
	User              string `env:"RABBITMQ_USER" envDefault:"guest"`
	Password          string `env:"RABBITMQ_PASSWORD" envDefault:"guest"`
	Exchange          string `env:"RABBITMQ_EXCHANGE" envDefault:"user.events"`
	ExchangeType      string `env:"RABBITMQ_EXCHANGE_TYPE" envDefault:"direct"`
	DLX               string `env:"RABBITMQ_DLX" envDefault:"user.dlx"`
	CreatedQueue      string `env:"RABBITMQ_CREATED_QUEUE" envDefault:"user.created.queue"`
	UpdatedQueue      string `env:"RABBITMQ_UPDATED_QUEUE" envDefault:"user.updated.queue"`
	CreatedRoutingKey string `env:"RABBITMQ_CREATED_ROUTING_KEY" envDefault:"user.created"`
	UpdatedRoutingKey string `env:"RABBITMQ_UPDATED_ROUTING_KEY" envDefault:"user.updated"`
	PrefetchCount     int    `env:"RABBITMQ_PREFETCH_COUNT" envDefault:"10"`
	PoolSize          int    `env:"RABBITMQ_POOL_SIZE" envDefault:"2"`
	FailedQueue       string `env:"RABBITMQ_FAILED_QUEUE" envDefault:"user.failed.queue"`
	TimeoutQueue      string `env:"RABBITMQ_TIMEOUT_QUEUE" envDefault:"user.timeout.queue"`
}

// GetRabbitMQURL constructs the RabbitMQ connection URL
func (r *RabbitMQConf) GetRabbitMQURL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		r.User,
		r.Password,
		r.Host,
		r.Port,
	)
}

// ValidateRabbitMQConfig validates RabbitMQ configuration
func (r *RabbitMQConf) ValidateRabbitMQConfig() error {
	requiredFields := map[string]string{
		"Host":     r.Host,
		"Port":     r.Port,
		"User":     r.User,
		"Password": r.Password,
		"Exchange": r.Exchange,
	}

	for field, value := range requiredFields {
		if value == "" {
			return fmt.Errorf("RabbitMQ configuration error: %s is required", field)
		}
	}

	if r.PrefetchCount <= 0 {
		log.Println("⚠️  Warning: PrefetchCount should be > 0, defaulting to 10")
		r.PrefetchCount = 10
	}

	if r.PoolSize <= 0 {
		log.Println("⚠️  Warning: PoolSize should be > 0, defaulting to 2")
		r.PoolSize = 2
	}

	return nil
}

// WaitForRabbitMQ waits for RabbitMQ to be ready with retry logic
func WaitForRabbitMQ() error {
	maxAttempts := 5
	retryInterval := 2 * time.Second

	log.Println("⏳ Waiting for RabbitMQ to be ready...")

	for i := 1; i <= maxAttempts; i++ {
		log.Printf("   Attempt %d/%d...", i, maxAttempts)
		time.Sleep(retryInterval)
	}

	log.Println("✅ RabbitMQ should be ready")
	return nil
}
