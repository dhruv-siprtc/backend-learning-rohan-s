package consumer

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

type ConsumerService struct {
	createdWorkerPool workerpool.Pool
	updatedWorkerPool workerpool.Pool
	rmqConfig         config.RabbitMQConf
}

// InitializeConsumer initializes Paota consumers for both queues
func InitializeConsumer(rmqConfig config.RabbitMQConf) (*ConsumerService, error) {
	if err := rmqConfig.ValidateRabbitMQConfig(); err != nil {
		return nil, fmt.Errorf("invalid RabbitMQ configuration: %w", err)
	}

	consumer := &ConsumerService{
		rmqConfig: rmqConfig,
	}

	// Initialize USER_CREATED consumer
	createdWorkerPool, err := consumer.initWorkerPool(
		rmqConfig.CreatedQueue,
		rmqConfig.CreatedRoutingKey,
		"user_created_consumer",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize USER_CREATED consumer: %w", err)
	}
	consumer.createdWorkerPool = createdWorkerPool

	// Initialize USER_UPDATED consumer
	updatedWorkerPool, err := consumer.initWorkerPool(
		rmqConfig.UpdatedQueue,
		rmqConfig.UpdatedRoutingKey,
		"user_updated_consumer",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize USER_UPDATED consumer: %w", err)
	}
	consumer.updatedWorkerPool = updatedWorkerPool

	log.Println("✅ Paota consumers initialized successfully")
	return consumer, nil
}

// initWorkerPool creates a worker pool for a specific queue
func (c *ConsumerService) initWorkerPool(queueName, routingKey, consumerTag string) (workerpool.Pool, error) {
	paotaConfig := paotaconfig.Config{
		Broker:        "amqp",
		TaskQueueName: queueName,
		AMQP: &paotaconfig.AMQPConfig{
			Url:                c.rmqConfig.GetRabbitMQURL(),
			Exchange:           c.rmqConfig.Exchange,
			ExchangeType:       "topic",
			BindingKey:         routingKey,
			PrefetchCount:      int(c.rmqConfig.PrefetchCount),
			ConnectionPoolSize: int(c.rmqConfig.PoolSize),
			DelayedQueue:       "",
			TimeoutQueue:       "",
			FailedQueue:        c.rmqConfig.DLX,
		},
	}

	workerPool, err := workerpool.NewWorkerPoolWithConfig(
		context.Background(),
		c.rmqConfig.PrefetchCount,
		consumerTag,
		paotaConfig,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool for %s: %w", queueName, err)
	}

	if workerPool == nil {
		return nil, fmt.Errorf("worker pool creation returned nil for %s", queueName)
	}

	return workerPool, nil
}

// Start starts consuming messages from both queues
func (c *ConsumerService) Start() error {
	// Register task handlers for both worker pools
	if err := c.registerTaskHandlers(); err != nil {
		return fmt.Errorf("failed to register task handlers: %w", err)
	}

	log.Printf("🎧 Starting USER_CREATED consumer for queue: %s", c.rmqConfig.CreatedQueue)
	log.Printf("🎧 Starting USER_UPDATED consumer for queue: %s", c.rmqConfig.UpdatedQueue)

	// Start USER_CREATED consumer in a goroutine (non-blocking)
	go func() {
		if err := c.createdWorkerPool.Start(); err != nil {
			log.Printf("❌ USER_CREATED consumer error: %v", err)
		}
	}()

	// Start USER_UPDATED consumer in main goroutine (blocking)
	// This will block until a shutdown signal is received
	if err := c.updatedWorkerPool.Start(); err != nil {
		return fmt.Errorf("failed to start USER_UPDATED consumer: %w", err)
	}

	return nil
}

// registerTaskHandlers registers handlers for different event types
func (c *ConsumerService) registerTaskHandlers() error {
	// Register USER_CREATED handler
	createdTasks := map[string]interface{}{
		"USER_CREATED": c.handleUserCreated,
	}
	if err := c.createdWorkerPool.RegisterTasks(createdTasks); err != nil {
		return fmt.Errorf("failed to register USER_CREATED handler: %w", err)
	}

	// Register USER_UPDATED handler
	updatedTasks := map[string]interface{}{
		"USER_UPDATED": c.handleUserUpdated,
	}
	if err := c.updatedWorkerPool.RegisterTasks(updatedTasks); err != nil {
		return fmt.Errorf("failed to register USER_UPDATED handler: %w", err)
	}

	return nil
}

// handleUserCreated processes USER_CREATED events
// Context parameter is required by Paota's task handler signature
func (c *ConsumerService) handleUserCreated(ctx context.Context, signature *schema.Signature) error {
	if len(signature.Args) == 0 {
		return fmt.Errorf("no arguments in signature")
	}

	eventJSON, ok := signature.Args[0].Value.(string)
	if !ok {
		return fmt.Errorf("invalid argument type, expected string")
	}

	var event events.UserEvent
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		log.Printf("❌ Failed to unmarshal USER_CREATED event: %v", err)
		return err // Message will be retried or sent to DLQ
	}

	// Simulate sending welcome email
	log.Printf("📧 [USER_CREATED] Welcome email sent to %s (UserID: %d)",
		event.Data.Email,
		event.Data.UserID,
	)

	// In production, this would call an actual email service
	// Example: emailService.SendWelcomeEmail(ctx, event.Data.Email, event.Data.Name)

	return nil // Message will be acknowledged
}

// handleUserUpdated processes USER_UPDATED events
// Context parameter is required by Paota's task handler signature
func (c *ConsumerService) handleUserUpdated(ctx context.Context, signature *schema.Signature) error {
	if len(signature.Args) == 0 {
		return fmt.Errorf("no arguments in signature")
	}

	eventJSON, ok := signature.Args[0].Value.(string)
	if !ok {
		return fmt.Errorf("invalid argument type, expected string")
	}

	var event events.UserEvent
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		log.Printf("❌ Failed to unmarshal USER_UPDATED event: %v", err)
		return err // Message will be retried or sent to DLQ
	}

	// Log user update for audit
	log.Printf("📝 [USER_UPDATED] User %d (%s) profile updated",
		event.Data.UserID,
		event.Data.Email,
	)

	// In production, this might:
	// - Update search indexes
	// - Invalidate caches
	// - Notify other services
	// - Log to audit trail

	return nil // Message will be acknowledged
}

// Close closes all consumer connections
func (c *ConsumerService) Close() error {
	log.Println("🔌 Stopping consumer worker pools...")
	c.createdWorkerPool.Stop()
	c.updatedWorkerPool.Stop()
	log.Println("✅ Consumer worker pools stopped")
	return nil
}
