package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"queue-microservice-case/shared/contracts"
)

type KafkaBroker struct {
	producer sarama.SyncProducer
	consumer sarama.ConsumerGroup
	config   *sarama.Config
	brokers  []string
}

// NewKafkaBroker creates a new Kafka broker instance
func NewKafkaBroker(brokers []string) (*KafkaBroker, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Version = sarama.V2_8_0_0

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	consumer, err := sarama.NewConsumerGroup(brokers, "default-group", config)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &KafkaBroker{
		producer: producer,
		consumer: consumer,
		config:   config,
		brokers:   brokers,
	}, nil
}

func (k *KafkaBroker) Publish(ctx context.Context, topic string, event *contracts.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(event.IdempotencyID),
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{Key: []byte("correlation_id"), Value: []byte(event.CorrelationID)},
			{Key: []byte("idempotency_id"), Value: []byte(event.IdempotencyID)},
			{Key: []byte("event_type"), Value: []byte(event.EventType)},
		},
	}

	partition, offset, err := k.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("Published event to Kafka: topic=%s, partition=%d, offset=%d, correlation_id=%s, idempotency_id=%s",
		topic, partition, offset, event.CorrelationID, event.IdempotencyID)

	return nil
}

func (k *KafkaBroker) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	consumer := &kafkaConsumerGroupHandler{
		topic:   topic,
		handler: handler,
	}

	go func() {
		for {
			if err := k.consumer.Consume(ctx, []string{topic}, consumer); err != nil {
				log.Printf("Error consuming from Kafka: %v", err)
				time.Sleep(5 * time.Second)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	return nil
}

func (k *KafkaBroker) PublishToDLQ(ctx context.Context, topic string, dlqEvent *DLQEvent) error {
	dlqTopic := topic + ".dlq"
	return k.Publish(ctx, dlqTopic, dlqEvent.OriginalEvent)
}

func (k *KafkaBroker) Close() error {
	if err := k.producer.Close(); err != nil {
		return err
	}
	return k.consumer.Close()
}

// kafkaConsumerGroupHandler implements sarama.ConsumerGroupHandler
type kafkaConsumerGroupHandler struct {
	topic   string
	handler MessageHandler
}

func (h *kafkaConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *kafkaConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *kafkaConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			var event contracts.Event
			if err := json.Unmarshal(message.Value, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				session.MarkMessage(message, "")
				continue
			}

			ctx := context.Background()
			if err := h.handler(ctx, &event); err != nil {
				log.Printf("Handler error for event %s: %v", event.EventID, err)
				// In production, implement retry logic here
				// For now, we mark as processed to avoid infinite loops
			}

			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

