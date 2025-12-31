package messaging

import (
	"context"
	"queue-microservice-case/shared/contracts"
)

// MessageBroker defines the interface for message brokers
// This abstraction allows switching between Kafka and RabbitMQ
// without changing the core application code
type MessageBroker interface {
	// Publish sends an event to a topic/queue
	Publish(ctx context.Context, topic string, event *contracts.Event) error

	// Subscribe starts consuming events from a topic/queue
	// The handler function will be called for each message
	// If the handler returns an error, the message will be retried or sent to DLQ
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error

	// PublishToDLQ sends a failed event to the Dead Letter Queue
	PublishToDLQ(ctx context.Context, topic string, dlqEvent *DLQEvent) error

	// Close gracefully closes the broker connection
	Close() error
}

// MessageHandler processes a single event
// Returns error if processing failed and should be retried/sent to DLQ
type MessageHandler func(ctx context.Context, event *contracts.Event) error

// DLQEvent represents an event that failed processing and is sent to DLQ
type DLQEvent struct {
	OriginalEvent *contracts.Event `json:"original_event"`
	Error         string           `json:"error"`
	RetryCount    int              `json:"retry_count"`
	LastAttempt   string           `json:"last_attempt"` // ISO-8601 timestamp
}

