package consumer

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

type ConsumerService struct {
	createdWorkerPool workerpool.Pool
	updatedWorkerPool workerpool.Pool
	rmqConfig         config.RabbitMQConf
	wg                sync.WaitGroup
	cancel            context.CancelFunc
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
			ExchangeType:       c.rmqConfig.ExchangeType,
			BindingKey:         routingKey,
			PrefetchCount:      int(c.rmqConfig.PrefetchCount),
			ConnectionPoolSize: int(c.rmqConfig.PoolSize),
			DelayedQueue:       "",
			TimeoutQueue:       "",
			FailedQueue:        c.rmqConfig.DLX,
		},
	}

	ctx := context.Background()

	// Number of worker goroutines - use PrefetchCount as the worker count

	workerPool, err := workerpool.NewWorkerPoolWithConfig(
		ctx,
		uint(c.rmqConfig.PrefetchCount), // Fixed: Pass number of workers as int
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

// Start starts consuming messages from both queues in separate goroutines
func (c *ConsumerService) Start() error {
	// Register task handlers
	if err := c.registerTaskHandlers(); err != nil {
		return fmt.Errorf("failed to register task handlers: %w", err)
	}

	log.Printf("🎧 Starting consumers...")
	log.Printf("   USER_CREATED queue: %s (routing key: %s)", c.rmqConfig.CreatedQueue, c.rmqConfig.CreatedRoutingKey)
	log.Printf("   USER_UPDATED queue: %s (routing key: %s)", c.rmqConfig.UpdatedQueue, c.rmqConfig.UpdatedRoutingKey)

	// Start USER_CREATED consumer in goroutine
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		log.Println("🎧 Starting USER_CREATED consumer...")
		if err := c.createdWorkerPool.Start(); err != nil {
			log.Printf("❌ USER_CREATED consumer error: %v", err)
		}
	}()

	// Start USER_UPDATED consumer in goroutine
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		log.Println("🎧 Starting USER_UPDATED consumer...")
		if err := c.updatedWorkerPool.Start(); err != nil {
			log.Printf("❌ USER_UPDATED consumer error: %v", err)
		}
	}()

	log.Println("✅ Both consumers are running")

	// Wait for all consumers to finish
	c.wg.Wait()
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

	log.Println("✅ Task handlers registered successfully")
	return nil
}

// handleUserCreated processes USER_CREATED events
func (c *ConsumerService) handleUserCreated(ctx context.Context, signature *schema.Signature) error {
	if len(signature.Args) == 0 {
		log.Printf("❌ [USER_CREATED] No arguments in signature")
		return fmt.Errorf("no arguments in signature")
	}

	eventJSON, ok := signature.Args[0].Value.(string)
	if !ok {
		log.Printf("❌ [USER_CREATED] Invalid argument type, expected string")
		return fmt.Errorf("invalid argument type, expected string")
	}

	var event events.UserEvent
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		log.Printf("❌ [USER_CREATED] Failed to unmarshal event: %v", err)
		return err
	}

	// Simulate sending welcome email
	log.Printf("📧 [USER_CREATED] Welcome email sent to %s (UserID: %d, Name: %s)",
		event.Data.Email,
		event.Data.UserID,
		event.Data.Name,
	)

	// In production, call actual email service here
	// Example: emailService.SendWelcomeEmail(ctx, event.Data.Email, event.Data.Name)

	return nil
}

// handleUserUpdated processes USER_UPDATED events
func (c *ConsumerService) handleUserUpdated(ctx context.Context, signature *schema.Signature) error {
	if len(signature.Args) == 0 {
		log.Printf("❌ [USER_UPDATED] No arguments in signature")
		return fmt.Errorf("no arguments in signature")
	}

	eventJSON, ok := signature.Args[0].Value.(string)
	if !ok {
		log.Printf("❌ [USER_UPDATED] Invalid argument type, expected string")
		return fmt.Errorf("invalid argument type, expected string")
	}

	var event events.UserEvent
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		log.Printf("❌ [USER_UPDATED] Failed to unmarshal event: %v", err)
		return err
	}

	// Log user update for audit
	log.Printf("📝 [USER_UPDATED] User %d (%s - %s) profile updated",
		event.Data.UserID,
		event.Data.Name,
		event.Data.Email,
	)

	// In production, this might:
	// - Update search indexes
	// - Invalidate caches
	// - Notify other services
	// - Log to audit trail

	return nil
}

// Close gracefully closes all consumer connections
func (c *ConsumerService) Close() error {
	log.Println("🔌 Stopping consumer worker pools...")

	// Stop both worker pools
	if c.createdWorkerPool != nil {
		c.createdWorkerPool.Stop()
	}
	if c.updatedWorkerPool != nil {
		c.updatedWorkerPool.Stop()
	}

	// Wait for all goroutines to finish
	c.wg.Wait()

	log.Println("✅ Consumer worker pools stopped")
	return nil
}
