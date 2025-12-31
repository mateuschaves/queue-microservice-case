package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"queue-microservice-case/shared/contracts"
	"queue-microservice-case/shared/database"
	"queue-microservice-case/shared/logger"
	"queue-microservice-case/shared/messaging"
)

const (
	serviceName = "message-processor"
	topicIn     = "message.created"
	topicOut    = "message.status.updated"
)

func main() {
	// Initialize logger
	appLogger := logger.NewLogger(serviceName)

	// Get database connection string
	dbConnStr := getDatabaseConnectionString()
	repo, err := database.NewRepository(dbConnStr)
	if err != nil {
		appLogger.Error("Failed to connect to database", "", "", err, nil)
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	// Initialize message broker
	broker, err := messaging.NewMessageBroker()
	if err != nil {
		appLogger.Error("Failed to initialize message broker", "", "", err, nil)
		log.Fatalf("Failed to initialize message broker: %v", err)
	}
	defer broker.Close()

	// Subscribe to message.created events
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = broker.Subscribe(ctx, topicIn, createMessageHandler(repo, broker, appLogger))
	if err != nil {
		appLogger.Error("Failed to subscribe to topic", "", "", err, nil)
		log.Fatalf("Failed to subscribe: %v", err)
	}

	appLogger.Info("Message processor started", "", "", map[string]interface{}{
		"topic_in":  topicIn,
		"topic_out": topicOut,
	})

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	appLogger.Info("Shutting down message processor", "", "", nil)
}

func createMessageHandler(repo *database.Repository, broker messaging.MessageBroker, appLogger *logger.Logger) messaging.MessageHandler {
	return func(ctx context.Context, event *contracts.Event) error {
		appLogger.Info("Received message.created event", event.CorrelationID, event.IdempotencyID, map[string]interface{}{
			"event_id": event.EventID,
		})

		// Idempotency check: verify if this idempotency_id was already processed
		msg, exists, err := repo.CreateOrGetMessage(
			event.IdempotencyID,
			event.CorrelationID,
			event.Payload,
		)
		if err != nil {
			appLogger.Error("Failed to check/create message", event.CorrelationID, event.IdempotencyID, err, nil)
			return fmt.Errorf("failed to check/create message: %w", err)
		}

		if exists && msg.Status != "pending" {
			appLogger.Info("Message already processed, skipping", event.CorrelationID, event.IdempotencyID, map[string]interface{}{
				"current_status": msg.Status,
			})
			return nil // Idempotent: message already processed
		}

		// Simulate processing
		appLogger.Info("Processing message", event.CorrelationID, event.IdempotencyID, nil)
		time.Sleep(100 * time.Millisecond) // Simulate work

		// Update status to processing
		err = repo.UpdateMessageStatus(
			event.IdempotencyID,
			event.CorrelationID,
			"processing",
			serviceName,
			event.EventID,
			nil,
		)
		if err != nil {
			appLogger.Error("Failed to update message status", event.CorrelationID, event.IdempotencyID, err, nil)
			return fmt.Errorf("failed to update status: %w", err)
		}

		// Simulate more processing
		time.Sleep(200 * time.Millisecond)

		// Update status to processed
		err = repo.UpdateMessageStatus(
			event.IdempotencyID,
			event.CorrelationID,
			"processed",
			serviceName,
			event.EventID,
			nil,
		)
		if err != nil {
			appLogger.Error("Failed to update message status", event.CorrelationID, event.IdempotencyID, err, nil)
			return fmt.Errorf("failed to update status: %w", err)
		}

		// Publish message.status.updated event
		statusPayload := map[string]interface{}{
			"idempotency_id": event.IdempotencyID,
			"status":         "processed",
			"processed_at":   time.Now().UTC().Format(time.RFC3339),
		}

		statusEvent := contracts.NewEvent(
			"message.status.updated",
			event.CorrelationID,
			event.IdempotencyID,
			serviceName,
			statusPayload,
		)

		err = broker.Publish(ctx, topicOut, statusEvent)
		if err != nil {
			appLogger.Error("Failed to publish status update", event.CorrelationID, event.IdempotencyID, err, nil)
			return fmt.Errorf("failed to publish status update: %w", err)
		}

		appLogger.Info("Message processed successfully", event.CorrelationID, event.IdempotencyID, map[string]interface{}{
			"status": "processed",
		})

		return nil
	}
}

func getDatabaseConnectionString() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "queue_case")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

