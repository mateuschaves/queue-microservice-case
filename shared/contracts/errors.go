package contracts

import "errors"

var (
	ErrMissingEventID       = errors.New("event_id is required")
	ErrMissingCorrelationID = errors.New("correlation_id is required")
	ErrMissingIdempotencyID = errors.New("idempotency_id is required")
	ErrMissingEventType     = errors.New("event_type is required")
	ErrMissingSourceService = errors.New("source_service is required")
	ErrMissingTimestamp     = errors.New("timestamp is required")
)

