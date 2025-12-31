package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type LogEntry struct {
	Level          string `json:"level"`
	Service        string `json:"service"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	IdempotencyID  string `json:"idempotency_id,omitempty"`
	Message        string `json:"message"`
	Timestamp      string `json:"timestamp"`
	AdditionalData map[string]interface{} `json:"additional_data,omitempty"`
}

type Logger struct {
	serviceName string
}

func NewLogger(serviceName string) *Logger {
	return &Logger{serviceName: serviceName}
}

func (l *Logger) log(level, message, correlationID, idempotencyID string, additionalData map[string]interface{}) {
	entry := LogEntry{
		Level:         level,
		Service:       l.serviceName,
		CorrelationID: correlationID,
		IdempotencyID: idempotencyID,
		Message:       message,
		Timestamp:     getTimestamp(),
		AdditionalData: additionalData,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	log.Println(string(jsonBytes))
}

func (l *Logger) Info(message, correlationID, idempotencyID string, additionalData map[string]interface{}) {
	l.log("INFO", message, correlationID, idempotencyID, additionalData)
}

func (l *Logger) Error(message, correlationID, idempotencyID string, err error, additionalData map[string]interface{}) {
	if additionalData == nil {
		additionalData = make(map[string]interface{})
	}
	if err != nil {
		additionalData["error"] = err.Error()
	}
	l.log("ERROR", message, correlationID, idempotencyID, additionalData)
}

func (l *Logger) Warn(message, correlationID, idempotencyID string, additionalData map[string]interface{}) {
	l.log("WARN", message, correlationID, idempotencyID, additionalData)
}

func (l *Logger) Debug(message, correlationID, idempotencyID string, additionalData map[string]interface{}) {
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		l.log("DEBUG", message, correlationID, idempotencyID, additionalData)
	}
}

func getTimestamp() string {
	// Use RFC3339 format (ISO-8601 compatible)
	return time.Now().UTC().Format(time.RFC3339)
}

