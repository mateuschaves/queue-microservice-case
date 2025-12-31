package messaging

import (
	"fmt"
	"os"
)

// NewMessageBroker creates a message broker based on MESSAGE_BROKER environment variable
// Supported values: "kafka", "rabbit" or "rabbitmq"
func NewMessageBroker() (MessageBroker, error) {
	brokerType := os.Getenv("MESSAGE_BROKER")
	if brokerType == "" {
		brokerType = "kafka" // default
	}

	switch brokerType {
	case "kafka":
		brokers := getKafkaBrokers()
		return NewKafkaBroker(brokers)
	case "rabbit", "rabbitmq":
		url := getRabbitMQURL()
		return NewRabbitMQBroker(url)
	default:
		return nil, fmt.Errorf("unsupported message broker: %s (supported: kafka, rabbit, rabbitmq)", brokerType)
	}
}

func getKafkaBrokers() []string {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return []string{"localhost:9092"}
	}
	// Simple split by comma - in production, use proper parsing
	return []string{brokers}
}

func getRabbitMQURL() string {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		return "amqp://guest:guest@localhost:5672/"
	}
	return url
}

