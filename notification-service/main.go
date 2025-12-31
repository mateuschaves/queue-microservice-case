package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"queue-microservice-case/shared/contracts"
	"queue-microservice-case/shared/logger"
	"queue-microservice-case/shared/messaging"
)

const (
	serviceName = "notification-service"
	topicIn     = "message.status.updated"
)

func main() {
	// Initialize logger
	appLogger := logger.NewLogger(serviceName)

	// Initialize message broker
	broker, err := messaging.NewMessageBroker()
	if err != nil {
		appLogger.Error("Failed to initialize message broker", "", "", err, nil)
		log.Fatalf("Failed to initialize message broker: %v", err)
	}
	defer broker.Close()

	// Subscribe to message.status.updated events
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = broker.Subscribe(ctx, topicIn, createNotificationHandler(appLogger))
	if err != nil {
		appLogger.Error("Failed to subscribe to topic", "", "", err, nil)
		log.Fatalf("Failed to subscribe: %v", err)
	}

	appLogger.Info("Notification service started", "", "", map[string]interface{}{
		"topic_in": topicIn,
	})

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	appLogger.Info("Shutting down notification service", "", "", nil)
}

func createNotificationHandler(appLogger *logger.Logger) messaging.MessageHandler {
	return func(ctx context.Context, event *contracts.Event) error {
		appLogger.Info("Received message.status.updated event", event.CorrelationID, event.IdempotencyID, map[string]interface{}{
			"event_id": event.EventID,
		})

		// Extract status from payload
		status, ok := event.Payload["status"].(string)
		if !ok {
			appLogger.Warn("Status not found in payload", event.CorrelationID, event.IdempotencyID, nil)
			return nil
		}

		// Simulate notification logic
		appLogger.Info("Sending notification", event.CorrelationID, event.IdempotencyID, map[string]interface{}{
			"status": status,
		})

		// Simulate notification delay
		time.Sleep(50 * time.Millisecond)

		appLogger.Info("Notification sent successfully", event.CorrelationID, event.IdempotencyID, map[string]interface{}{
			"status": status,
		})

		return nil
	}
}

