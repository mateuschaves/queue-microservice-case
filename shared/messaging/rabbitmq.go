package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
	"queue-microservice-case/shared/contracts"
)

type RabbitMQBroker struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
}

// NewRabbitMQBroker creates a new RabbitMQ broker instance
func NewRabbitMQBroker(url string) (*RabbitMQBroker, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &RabbitMQBroker{
		conn:    conn,
		channel: channel,
		url:     url,
	}, nil
}

func (r *RabbitMQBroker) Publish(ctx context.Context, queue string, event *contracts.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	// Declare queue
	_, err := r.channel.QueueDeclare(
		queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = r.channel.Publish(
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
			Headers: amqp.Table{
				"correlation_id": event.CorrelationID,
				"idempotency_id": event.IdempotencyID,
				"event_type":     event.EventType,
			},
			MessageId:    event.EventID,
			Timestamp:    time.Now(),
			CorrelationId: event.CorrelationID,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published event to RabbitMQ: queue=%s, correlation_id=%s, idempotency_id=%s",
		queue, event.CorrelationID, event.IdempotencyID)

	return nil
}

func (r *RabbitMQBroker) Subscribe(ctx context.Context, queue string, handler MessageHandler) error {
	// Declare queue
	_, err := r.channel.QueueDeclare(
		queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange": "dlx",
		},
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Declare DLX
	err = r.channel.ExchangeDeclare(
		"dlx",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLX: %w", err)
	}

	// Declare DLQ
	dlqName := queue + ".dlq"
	_, err = r.channel.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ to DLX
	err = r.channel.QueueBind(
		dlqName,
		queue,
		"dlx",
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind DLQ: %w", err)
	}

	// Set QoS
	err = r.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := r.channel.Consume(
		queue,
		"",    // consumer
		false, // auto-ack (manual ack for retry logic)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					log.Println("RabbitMQ channel closed")
					return
				}

				var event contracts.Event
				if err := json.Unmarshal(msg.Body, &event); err != nil {
					log.Printf("Failed to unmarshal event: %v", err)
					msg.Nack(false, false) // Reject without requeue
					continue
				}

				ctx := context.Background()
				if err := handler(ctx, &event); err != nil {
					log.Printf("Handler error for event %s: %v", event.EventID, err)
					// Nack with requeue for retry, or send to DLQ after max retries
					// For simplicity, we'll reject without requeue (goes to DLQ)
					msg.Nack(false, false)
				} else {
					msg.Ack(false)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (r *RabbitMQBroker) PublishToDLQ(ctx context.Context, queue string, dlqEvent *DLQEvent) error {
	dlqQueue := queue + ".dlq"
	return r.Publish(ctx, dlqQueue, dlqEvent.OriginalEvent)
}

func (r *RabbitMQBroker) Close() error {
	if err := r.channel.Close(); err != nil {
		return err
	}
	return r.conn.Close()
}

