package contracts

import "time"

// Event represents the standard event contract used across all microservices
type Event struct {
	EventID        string                 `json:"event_id"`
	CorrelationID  string                 `json:"correlation_id"`
	IdempotencyID  string                 `json:"idempotency_id"`
	EventType      string                 `json:"event_type"`
	SourceService  string                 `json:"source_service"`
	Timestamp      string                 `json:"timestamp"` // ISO-8601 format
	Payload        map[string]interface{} `json:"payload"`
}

// NewEvent creates a new event with required fields
func NewEvent(eventType, correlationID, idempotencyID, sourceService string, payload map[string]interface{}) *Event {
	return &Event{
		EventID:       generateEventID(),
		CorrelationID: correlationID,
		IdempotencyID: idempotencyID,
		EventType:     eventType,
		SourceService: sourceService,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Payload:       payload,
	}
}

// Validate ensures all required fields are present
func (e *Event) Validate() error {
	if e.EventID == "" {
		return ErrMissingEventID
	}
	if e.CorrelationID == "" {
		return ErrMissingCorrelationID
	}
	if e.IdempotencyID == "" {
		return ErrMissingIdempotencyID
	}
	if e.EventType == "" {
		return ErrMissingEventType
	}
	if e.SourceService == "" {
		return ErrMissingSourceService
	}
	if e.Timestamp == "" {
		return ErrMissingTimestamp
	}
	return nil
}

