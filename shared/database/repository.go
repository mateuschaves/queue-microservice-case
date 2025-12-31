package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type Message struct {
	IdempotencyID string                 `json:"idempotency_id"`
	CorrelationID  string                 `json:"correlation_id"`
	Status       string                 `json:"status"`
	Payload      map[string]interface{} `json:"payload"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type MessageHistory struct {
	ID            int       `json:"id"`
	IdempotencyID string    `json:"idempotency_id"`
	CorrelationID string    `json:"correlation_id"`
	Status        string    `json:"status"`
	ServiceName   string    `json:"service_name"`
	EventID       string    `json:"event_id"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(connectionString string) (*Repository, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Repository{db: db}, nil
}

// CreateOrGetMessage creates a message or returns existing one (idempotency check)
func (r *Repository) CreateOrGetMessage(idempotencyID, correlationID string, payload map[string]interface{}) (*Message, bool, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal payload: %w", err)
	}

	var msg Message
	var exists bool

	query := `
		INSERT INTO messages (idempotency_id, correlation_id, status, payload, created_at, updated_at)
		VALUES ($1, $2, 'pending', $3, NOW(), NOW())
		ON CONFLICT (idempotency_id) DO UPDATE SET updated_at = NOW()
		RETURNING idempotency_id, correlation_id, status, payload, created_at, updated_at
	`

	var payloadBytes []byte
	err = r.db.QueryRow(query, idempotencyID, correlationID, payloadJSON).Scan(
		&msg.IdempotencyID,
		&msg.CorrelationID,
		&msg.Status,
		&payloadBytes,
		&msg.CreatedAt,
		&msg.UpdatedAt,
	)

	if err != nil {
		// Check if it's a conflict (already exists)
		if err == sql.ErrNoRows {
			// Try to get existing message
			query = `SELECT idempotency_id, correlation_id, status, payload, created_at, updated_at 
					 FROM messages WHERE idempotency_id = $1`
			err = r.db.QueryRow(query, idempotencyID).Scan(
				&msg.IdempotencyID,
				&msg.CorrelationID,
				&msg.Status,
				&payloadBytes,
				&msg.CreatedAt,
				&msg.UpdatedAt,
			)
			if err == nil {
				exists = true
			}
		}
		if !exists {
			return nil, false, fmt.Errorf("failed to create/get message: %w", err)
		}
	}

	// Unmarshal payload
	if err := json.Unmarshal(payloadBytes, &msg.Payload); err != nil {
		return nil, exists, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &msg, exists, nil
}

// UpdateMessageStatus updates message status and creates history entry
func (r *Repository) UpdateMessageStatus(idempotencyID, correlationID, status, serviceName, eventID string, errorMsg *string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update message
	updateQuery := `UPDATE messages SET status = $1, updated_at = NOW() WHERE idempotency_id = $2`
	_, err = tx.Exec(updateQuery, status, idempotencyID)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	// Insert history
	historyQuery := `
		INSERT INTO message_history (idempotency_id, correlation_id, status, service_name, event_id, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err = tx.Exec(historyQuery, idempotencyID, correlationID, status, serviceName, eventID, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to insert history: %w", err)
	}

	return tx.Commit()
}

// GetMessage retrieves a message by idempotency_id
func (r *Repository) GetMessage(idempotencyID string) (*Message, error) {
	var msg Message
	query := `SELECT idempotency_id, correlation_id, status, payload, created_at, updated_at 
			  FROM messages WHERE idempotency_id = $1`

	var payloadBytes []byte
	err := r.db.QueryRow(query, idempotencyID).Scan(
		&msg.IdempotencyID,
		&msg.CorrelationID,
		&msg.Status,
		&payloadBytes,
		&msg.CreatedAt,
		&msg.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	if err := json.Unmarshal(payloadBytes, &msg.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &msg, nil
}

// GetMessageHistory retrieves all history entries for a message
func (r *Repository) GetMessageHistory(idempotencyID string) ([]MessageHistory, error) {
	query := `
		SELECT id, idempotency_id, correlation_id, status, service_name, event_id, error_message, created_at
		FROM message_history
		WHERE idempotency_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, idempotencyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var history []MessageHistory
	for rows.Next() {
		var h MessageHistory
		err := rows.Scan(
			&h.ID,
			&h.IdempotencyID,
			&h.CorrelationID,
			&h.Status,
			&h.ServiceName,
			&h.EventID,
			&h.ErrorMessage,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		history = append(history, h)
	}

	return history, nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}

