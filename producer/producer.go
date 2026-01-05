package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"golang-postgre/config"
	"golang-postgre/events"

	paotaconfig "github.com/surendratiwari3/paota/config"
	"github.com/surendratiwari3/paota/schema"
	"github.com/surendratiwari3/paota/workerpool"
)

type ProducerService struct {
	workerPool *workerpool.Pool
	rmqConfig  config.RabbitMQConf
}

var producerInstance *ProducerService

// InitializeProducer initializes the Paota producer
func InitializeProducer(rmqConfig config.RabbitMQConf) (*ProducerService, error) {
	if producerInstance != nil {
		return producerInstance, nil
	}

	if err := rmqConfig.ValidateRabbitMQConfig(); err != nil {
		return nil, fmt.Errorf("invalid RabbitMQ configuration: %w", err)
	}

	paotaConfig := paotaconfig.Config{
		Broker:        "amqp",
		TaskQueueName: "user.producer.queue", // Producer queue name
		AMQP: &paotaconfig.AMQPConfig{
			Url:                rmqConfig.GetRabbitMQURL(),
			Exchange:           rmqConfig.Exchange,
			ExchangeType:       "topic",
			BindingKey:         "",
			PrefetchCount:      int(rmqConfig.PrefetchCount),
			ConnectionPoolSize: int(rmqConfig.PoolSize),
			DelayedQueue:       "",
			TimeoutQueue:       "",
			FailedQueue:        "",
		},
	}

	workerPool, err := workerpool.NewWorkerPoolWithConfig(
		context.Background(),
		rmqConfig.PoolSize,
		"user.producer",
		paotaConfig,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize producer worker pool: %w", err)
	}

	if workerPool == nil {
		return nil, fmt.Errorf("worker pool initialization returned nil")
	}

	producerInstance = &ProducerService{
		workerPool: &workerPool,
		rmqConfig:  rmqConfig,
	}

	log.Println("✅ Paota producer initialized successfully")
	return producerInstance, nil
}

// GetProducer returns the singleton producer instance
func GetProducer() (*ProducerService, error) {
	if producerInstance == nil {
		return nil, fmt.Errorf("producer not initialized, call InitializeProducer first")
	}
	return producerInstance, nil
}

// PublishUserCreated publishes USER_CREATED event
func (p *ProducerService) PublishUserCreated(event events.UserEvent) error {
	return p.publishEvent(event, p.rmqConfig.CreatedRoutingKey, "USER_CREATED")
}

// PublishUserUpdated publishes USER_UPDATED event
func (p *ProducerService) PublishUserUpdated(event events.UserEvent) error {
	return p.publishEvent(event, p.rmqConfig.UpdatedRoutingKey, "USER_UPDATED")
}

// publishEvent publishes an event to RabbitMQ using Paota
func (p *ProducerService) publishEvent(event events.UserEvent, routingKey, eventType string) error {
	if p.workerPool == nil {
		return fmt.Errorf("worker pool not initialized")
	}

	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create task signature
	signature := &schema.Signature{
		Name:       eventType,
		RoutingKey: routingKey,
		Args: []schema.Arg{
			{
				Type:  "string",
				Value: string(eventJSON),
			},
		},
		RetryCount:   3,
		RetryTimeout: 30,
	}

	// Send task asynchronously using SendTaskWithContext
	state, err := (*p.workerPool).SendTaskWithContext(context.Background(), signature)
	if err != nil {
		return fmt.Errorf("failed to send %s event: %w", eventType, err)
	}

	// Check if the task was queued successfully
	if state != nil && state.Status == "Pending" {
		log.Printf("✅ Event %s published successfully (UserID: %d, TaskID: %s)",
			eventType, event.Data.UserID, state.Request.UUID)
		return nil
	}

	log.Printf("⚠️ Event %s published with status: %s (UserID: %d)",
		eventType, state.Status, event.Data.UserID)
	return nil
}

// Close closes the producer connection
func (p *ProducerService) Close() error {
	if p.workerPool != nil {
		log.Println("🔌 Closing producer worker pool...")
		(*p.workerPool).Stop()
		return nil
	}
	return nil
}
