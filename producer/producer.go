package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"golang-postgre/config"
	"golang-postgre/events"

	paotaconfig "github.com/surendratiwari3/paota/config"
	"github.com/surendratiwari3/paota/schema"
	"github.com/surendratiwari3/paota/workerpool"
)

type ProducerService struct {
	createdPool workerpool.Pool
	updatedPool workerpool.Pool
	rmqConfig   config.RabbitMQConf
	mu          sync.RWMutex
}

var (
	producerInstance *ProducerService
	producerOnce     sync.Once
)

// InitializeProducer initializes separate producer pools for each event type
func InitializeProducer(rmqConfig config.RabbitMQConf) (*ProducerService, error) {
	var initErr error

	producerOnce.Do(func() {
		if err := rmqConfig.ValidateRabbitMQConfig(); err != nil {
			initErr = fmt.Errorf("invalid RabbitMQ configuration: %w", err)
			return
		}

		producer := &ProducerService{
			rmqConfig: rmqConfig,
		}

		// Initialize USER_CREATED producer pool
		createdPool, err := producer.initProducerPool(
			rmqConfig.CreatedQueue,
			rmqConfig.CreatedRoutingKey,
			"user_created_producer",
		)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize USER_CREATED producer: %w", err)
			return
		}
		producer.createdPool = createdPool

		// Initialize USER_UPDATED producer pool
		updatedPool, err := producer.initProducerPool(
			rmqConfig.UpdatedQueue,
			rmqConfig.UpdatedRoutingKey,
			"user_updated_producer",
		)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize USER_UPDATED producer: %w", err)
			return
		}
		producer.updatedPool = updatedPool

		producerInstance = producer
		log.Println("✅ Paota producers initialized successfully")
	})

	if initErr != nil {
		return nil, initErr
	}

	return producerInstance, nil
}

// initProducerPool creates a producer pool for a specific queue
func (p *ProducerService) initProducerPool(queueName, routingKey, tag string) (workerpool.Pool, error) {
	paotaConfig := paotaconfig.Config{
		Broker:        "amqp",
		TaskQueueName: queueName,
		AMQP: &paotaconfig.AMQPConfig{
			Url:                p.rmqConfig.GetRabbitMQURL(),
			Exchange:           p.rmqConfig.Exchange,
			ExchangeType:       p.rmqConfig.ExchangeType,
			BindingKey:         routingKey,
			PrefetchCount:      int(p.rmqConfig.PrefetchCount),
			ConnectionPoolSize: int(p.rmqConfig.PoolSize),
			DelayedQueue:       "",
			TimeoutQueue:       "",
			FailedQueue:        p.rmqConfig.DLX,
		},
	}
	ctx := context.Background()
	// Pass nil as context - Paota will handle context internally
	workerPool, err := workerpool.NewWorkerPoolWithConfig(
		ctx,
		1, // Single worker for producer
		tag,
		paotaConfig,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create producer pool for %s: %w", queueName, err)
	}

	if workerPool == nil {
		return nil, fmt.Errorf("producer pool creation returned nil for %s", queueName)
	}

	return workerPool, nil
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
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.createdPool == nil {
		return fmt.Errorf("USER_CREATED producer pool not initialized")
	}

	return p.publishEvent(p.createdPool, event, p.rmqConfig.CreatedRoutingKey, "USER_CREATED")
}

// PublishUserUpdated publishes USER_UPDATED event
func (p *ProducerService) PublishUserUpdated(event events.UserEvent) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.updatedPool == nil {
		return fmt.Errorf("USER_UPDATED producer pool not initialized")
	}

	return p.publishEvent(p.updatedPool, event, p.rmqConfig.UpdatedRoutingKey, "USER_UPDATED")
}

// publishEvent publishes an event to RabbitMQ using Paota
func (p *ProducerService) publishEvent(pool workerpool.Pool, event events.UserEvent, routingKey, eventType string) error {
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

	// Send task asynchronously
	state, err := pool.SendTaskWithContext(context.Background(), signature)
	if err != nil {
		log.Printf("❌ Failed to send %s event: %v", eventType, err)
		return fmt.Errorf("failed to send %s event: %w", eventType, err)
	}

	if state != nil {
		log.Printf("✅ [%s] Event published successfully (UserID: %d, Email: %s, TaskID: %s, Status: %s)",
			eventType, event.Data.UserID, event.Data.Email, state.Request.UUID, state.Status)
	} else {
		log.Printf("⚠️ [%s] Event published but state is nil (UserID: %d)", eventType, event.Data.UserID)
	}

	return nil
}

// Close closes all producer connections
func (p *ProducerService) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Println("🔌 Closing producer pools...")

	if p.createdPool != nil {
		p.createdPool.Stop()
	}

	if p.updatedPool != nil {
		p.updatedPool.Stop()
	}

	log.Println("✅ Producer pools closed")
	return nil
}
